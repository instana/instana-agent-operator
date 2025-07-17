/*
 * (c) Copyright IBM Corp. 2024
 * (c) Copyright Instana Inc. 2024
 */

package e2e

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

	securityv1 "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1"
	log "k8s.io/klog/v2"

	v1 "github.com/instana/instana-agent-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	e2etypes "sigs.k8s.io/e2e-framework/pkg/types"
	"sigs.k8s.io/e2e-framework/support/utils"
)

// This file exposes the reusable assets which are used during the e2e test

// env.Funcs to be used in the test initialization

// DeleteAgentNamespace ensures a proper cleanup of existing instana agent installations.
// The namespace cannot be just deleted in all scenarios, as finalizers on the agent CR might block the namespace termination
func EnsureAgentNamespaceDeletion() env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		log.Info("==== Startup Cleanup, errors are expected if resources are not available ====")
		log.Infof("Ensure namespace %s is not present", cfg.Namespace())
		// Create a client to interact with the Kube API
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			return ctx, fmt.Errorf("failed to initialize client: %v", err)
		}

		p := utils.RunCommand("kubectl get pods -n instana-agent")
		log.Info("Current pods: ", p.Command(), p.ExitCode(), "\n", p.Result())

		p = utils.RunCommand("kubectl get agent instana-agent -o yaml -n instana-agent")
		// redact agent key if present
		log.Info("Current agent CR: ", p.Command(), p.ExitCode(), "\n", strings.ReplaceAll(p.Result(), InstanaTestCfg.InstanaBackend.AgentKey, "***"))

		// Cleanup a potentially existing Agent CR first
		if _, err = DeleteAgentCRIfPresent()(ctx, cfg); err != nil {
			log.Info("Agent CR cleanup err: ", err)
		}

		log.Info("Agent CR cleanup completed")

		// full purge of resources if anything would be left in the cluster
		p = utils.RunCommand("kubectl delete crd/agents.instana.io clusterrole/instana-agent-k8sensor clusterrole/instana-agent-clusterrole clusterrole/leader-election-role clusterrolebinding/leader-election-rolebinding clusterrolebinding/instana-agent-clusterrolebinding")
		if p.Err() != nil {
			log.Warningf("Could not remove some artifacts, ignoring as they might not be present %s - %s - %s - %d", p.Command(), p.Err(), p.Out(), p.ExitCode())
		}

		// Check if namespace exist, otherwise just skip over it
		agentNamespace := &corev1.Namespace{}
		err = r.Get(ctx, InstanaNamespace, InstanaNamespace, agentNamespace)
		if errors.IsNotFound(err) {
			log.Infof("Namespace %s was not found, skipping deletion", cfg.Namespace())
			return ctx, nil
		}

		// Something on the API request failed, this should fail the cleanup
		if err != nil {
			return ctx, fmt.Errorf("failed to get namespace: %v", err)
		}

		// Delete the Namespace
		log.Info("Deleting namespace and waiting for successful termination")
		if err = r.Delete(ctx, agentNamespace); err != nil {
			return ctx, fmt.Errorf("namespace deletion failed: %v", err)
		}

		// Wait for the termination of the namespace
		namespaceList := &corev1.NamespaceList{
			Items: []corev1.Namespace{
				*agentNamespace,
			},
		}

		err = wait.For(conditions.New(r).ResourcesDeleted(namespaceList))
		if err != nil {
			return ctx, fmt.Errorf("error while waiting for namespace deletion: %v", err)
		}
		log.Infof("Namespace %s is gone", cfg.Namespace())
		log.Info("==== Cleanup compleated ====")
		return ctx, nil
	}
}

func DeleteAgentCRIfPresent() env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		log.Info("Ensure agent CR is not present")
		// Create a client to interact with the Kube API
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			return ctx, fmt.Errorf("cleanup: Error initializing client to delete agent CR: %v", err)
		}

		// Assume an existing namespace at this point, check if an agent CR is present (requires to adjust schema of current client)
		r.WithNamespace(InstanaNamespace)
		err = v1.AddToScheme(r.GetScheme())
		if err != nil {
			// If this fails, the cleanup will not work properly -> failing
			return ctx, fmt.Errorf("cleanup: Error could not add agent types to current scheme: %v", err)
		}

		// If the agent cr is available, but the operator is already gone, the finalizer will never be removed
		// This will lead to a delayed namespace termination which never completes. To avoid that, patch the agent CR
		// to remove the finalizer. Afterwards, it can be successfully deleted.
		agent := &v1.InstanaAgent{}
		err = r.Get(ctx, AgentCustomResourceName, InstanaNamespace, agent)
		if errors.IsNotFound(err) {
			// No agent cr found, skip this cleanup step
			log.Info("No agent CR present, skipping deletion")
			return ctx, nil
		}

		// The agent CR could not be fetched due to a different reason, failing
		if err != nil {
			return ctx, fmt.Errorf("cleanup: Fetch agent CR failed: %v", err)
		}

		// Removing the finalizer from the existing Agent CR to make it deletable
		// kubectl patch agent instana-agent -p '{"metadata":{"finalizers":[]}}' --type=merge
		log.Info("Patching agent cr to remove finalizers")
		err = r.Patch(ctx, agent, k8s.Patch{
			PatchType: types.MergePatchType,
			Data:      []byte(`{"metadata":{"finalizers":[]}}`),
		})
		if err != nil {
			return ctx, fmt.Errorf("cleanup: Patch agent CR failed: %v", err)
		}

		log.Info("Deleting CR")
		// delete explicitly, namespace deletion would delete the agent CR as well if the finalizer is not present
		err = r.Delete(ctx, agent)

		if err != nil {
			// The deletion failed for some reason, failing the cleanup
			return ctx, fmt.Errorf("cleanup: Delete agent CR failed: %v", err)
		}

		agentCrList := &v1.InstanaAgentList{
			Items: []v1.InstanaAgent{*agent},
		}

		// Ensure to wait for the agent CR to disappear before continuing
		err = wait.For(conditions.New(r).ResourcesDeleted(agentCrList))
		if err != nil {
			return ctx, fmt.Errorf("cleanup: Waiting for agent CR deletion failed: %v", err)
		}
		log.Info("Agent CR is gone")
		return ctx, nil
	}
}

// On OpenShift we need to ensure the instana-agent service account gets permission to the privilged security context
// This action is only necessary once per OCP cluster as it is not tight to a namespace, but to a cluster
func AdjustOcpPermissionsIfNecessary() env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		// Create a client to interact with the Kube API
		clientSet, err := kubernetes.NewForConfig(cfg.Client().RESTConfig())
		if err != nil {
			return ctx, fmt.Errorf("error creating a clientset: %v", err)
		}

		discoveryClient := discovery.NewDiscoveryClient(clientSet.RESTClient())
		apiGroups, err := discoveryClient.ServerGroups()
		if err != nil {
			return ctx, fmt.Errorf("failed to fetch apiGroups: %v", err)
		}

		isOpenShift := false
		for _, group := range apiGroups.Groups {
			if group.Name == "apps.openshift.io" {
				isOpenShift = true
				break
			}
		}

		if isOpenShift {
			command := "oc adm policy add-scc-to-user privileged -z instana-agent -n instana-agent"
			log.Infof("OpenShift detected, adding instana-agent service account to SecurityContextConstraints via api, command would be: %s\n", command)

			// replaced command execution with SDK call to not require `oc` cli
			securityClient, err := securityv1.NewForConfig(cfg.Client().RESTConfig())
			if err != nil {
				return ctx, fmt.Errorf("could not initialize securityClient: %v", err)
			}

			// get security context constraints
			scc, err := securityClient.SecurityContextConstraints().Get(ctx, "privileged", metav1.GetOptions{})
			if err != nil {
				return ctx, fmt.Errorf("failed to get SecurityContextContraints: %v", err)
			}

			// check if service account user is already listed in the scc
			serviceAccountId := fmt.Sprintf("system:serviceaccount:%s:%s", InstanaNamespace, "instana-agent")
			userFound := false

			for _, user := range scc.Users {
				if user == serviceAccountId {
					userFound = true
					break
				}
			}

			if userFound {
				log.Infof("Security Context Constraint \"privileged\" already lists service account user: %v\n", serviceAccountId)
				return ctx, nil
			}

			// updating Security Context Constraints to list instana service account
			scc.Users = append(scc.Users, serviceAccountId)

			_, err = securityClient.SecurityContextConstraints().Update(ctx, scc, metav1.UpdateOptions{})
			if err != nil {
				return ctx, fmt.Errorf("could not update Security Context Constraints on OCP cluster: %v", err)
			}

			return ctx, nil
		} else {
			// non-ocp environments do not require changes in the Security Context Constraints
			log.Info("Cluster is not an OpenShift cluster, no need to adjust the security context constraints")
		}
		return ctx, nil
	}
}

// Setup functions
func SetupOperatorDevBuild() e2etypes.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		// Create pull secret for custom registry
		t.Logf("Creating custom pull secret for %s", InstanaTestCfg.ContainerRegistry.Host)
		p := utils.RunCommand(
			fmt.Sprintf("kubectl create secret -n %s docker-registry %s --docker-server=%s --docker-username=%s --docker-password=%s",
				cfg.Namespace(),
				InstanaTestCfg.ContainerRegistry.Name,
				InstanaTestCfg.ContainerRegistry.Host,
				InstanaTestCfg.ContainerRegistry.User,
				InstanaTestCfg.ContainerRegistry.Password),
		)
		if p.Err() != nil {
			t.Fatal("Error while creating pull secret", p.Err(), p.Out(), p.ExitCode())
		}
		t.Log("Pull secret created")

		// Use make logic to ensure that local dev commands and test commands are in sync
		cmd := fmt.Sprintf("bash -c 'cd .. && IMG=%s:%s make install deploy'", InstanaTestCfg.OperatorImage.Name, InstanaTestCfg.OperatorImage.Tag)
		t.Logf("Deploy new dev build by running: %s", cmd)
		p = utils.RunCommand(cmd)
		if p.Err() != nil {
			t.Fatal("Error while deploying custom operator build during update installation", p.Command(), p.Err(), p.Out(), p.ExitCode())
		}
		t.Log("Deployment submitted")

		// Inject image pull secret into deployment, ensure to scale to 0 replicas and back to 2 replicas, otherwise pull secrets are not propagated correctly
		t.Log("Patch instana operator deployment to redeploy pods with image pull secret")
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal("Cleanup: Error initializing client", err)
		}
		r.WithNamespace(cfg.Namespace())
		agent := &appsv1.Deployment{}
		err = r.Get(ctx, InstanaOperatorDeploymentName, cfg.Namespace(), agent)
		replicas := agent.Spec.Replicas
		if err != nil {
			t.Fatal("Failed to get deployment-manager deployment", err)
		}
		err = r.Patch(ctx, agent, k8s.Patch{
			PatchType: types.MergePatchType,
			Data:      []byte(fmt.Sprintf(`{"spec":{ "replicas": 0, "template":{"spec": {"imagePullSecrets": [{"name": "%s"}]}}}}`, InstanaTestCfg.ContainerRegistry.Name)),
		})
		if err != nil {
			t.Fatal("Failed to patch deployment to include pull secret and 0 replicas", err)
		}

		err = r.Patch(ctx, agent, k8s.Patch{
			PatchType: types.MergePatchType,
			Data:      []byte(fmt.Sprintf(`{"spec":{ "replicas": %d }}`, *replicas)),
		})
		if err != nil {
			t.Fatal("Failed to patch deployment to include pull secret and 0 replicas", err)
		}
		t.Log("Patching completed")
		return ctx
	}
}

func DeployAgentCr(agent *v1.InstanaAgent) e2etypes.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
		}
		t.Log("Creating a new Agent CR")

		// Create Agent CR
		r := client.Resources(cfg.Namespace())
		err = v1.AddToScheme(r.GetScheme())
		if err != nil {
			t.Fatal("Could not add Agent CR to client scheme", err)
		}

		err = r.Create(ctx, agent)
		if err != nil {
			t.Fatal("Could not create Agent CR", err)
		}

		return ctx
	}
}

func UpdateAgentCr(agent *v1.InstanaAgent) e2etypes.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
		}
		t.Log("Creating a new Agent CR")

		// Create Agent CR
		r := client.Resources(cfg.Namespace())
		err = v1.AddToScheme(r.GetScheme())
		if err != nil {
			t.Fatal("Could not add Agent CR to client scheme", err)
		}

		// First get the current resource
		existingAgent := &v1.InstanaAgent{}
		err = r.Get(ctx, agent.Name, cfg.Namespace(), existingAgent)
		if err != nil {
			t.Fatal("Could not get existing Agent CR", err)
		}

		// Update the existing resource
		existingAgent.Spec = agent.Spec
		err = r.Update(ctx, existingAgent)
		if err != nil {
			t.Fatal("Could not update Agent CR", err)
		}

		return ctx
	}
}

// Assess functions
func WaitForDeploymentToBecomeReady(name string) e2etypes.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Logf("Waiting for deployment %s to become ready", name)
		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
		}
		dep := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: cfg.Namespace()},
		}

		// active wait for deployment to be created by the operator, if it is not coming up within 1 minute, something is really off
		for range 12 {
			err = client.Resources().Get(ctx, name, cfg.Namespace(), &dep)
			if err != nil {
				t.Log("Give the operator a few more seconds to inject resources")
				time.Sleep(5 * time.Second)
			} else {
				t.Logf("Deployment %s was present", name)
				break
			}
		}

		// wait for operator pods of the deployment to become ready
		err = wait.For(conditions.New(client.Resources()).DeploymentConditionMatch(&dep, appsv1.DeploymentAvailable, corev1.ConditionTrue), wait.WithTimeout(time.Minute*2))
		if err != nil {
			PrintOperatorLogs(ctx, cfg, t)
			t.Fatal(err)
		}
		t.Logf("Deployment %s is ready", name)
		return ctx
	}
}

// optional argument for the custom daemons set name
func WaitForAgentDaemonSetToBecomeReady(args ...string) e2etypes.StepFunc {
	var daemonSetName string
	if (len(args)) > 0 && args[0] != "" {
		daemonSetName = args[0]
	} else {
		daemonSetName = AgentDaemonSetName
	}
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Logf("Waiting for DaemonSet %s is ready", daemonSetName)
		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
		}
		ds := appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{Name: daemonSetName, Namespace: cfg.Namespace()},
		}
		err = wait.For(conditions.New(client.Resources()).DaemonSetReady(&ds), wait.WithTimeout(time.Minute*5))
		if err != nil {
			PrintOperatorLogs(ctx, cfg, t)
			t.Fatal(err)
		}
		t.Logf("DaemonSet %s is ready", daemonSetName)
		return ctx
	}
}

func EnsureOldControllerManagerDeploymentIsNotRunning() e2etypes.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Logf("Ensuring the old deployment %s is not running", InstanaOperatorOldDeploymentName)
		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
		}
		dep := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: InstanaOperatorOldDeploymentName, Namespace: cfg.Namespace()},
		}
		err = wait.For(conditions.New(client.Resources()).ResourceDeleted(&dep), wait.WithTimeout(time.Minute*2))
		if err != nil {
			PrintOperatorLogs(ctx, cfg, t)
			t.Fatal(err)
		}
		t.Logf("Deployment %s is deleted", InstanaOperatorOldDeploymentName)
		return ctx
	}
}

func EnsureOldClusterRoleIsGone() e2etypes.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Logf("Ensuring the old clusterrole %s is not running", InstanaOperatorOldClusterRoleName)
		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
		}
		clusterrole := rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{Name: InstanaOperatorOldClusterRoleName},
		}
		err = wait.For(conditions.New(client.Resources()).ResourceDeleted(&clusterrole), wait.WithTimeout(time.Minute*2))
		if err != nil {
			PrintOperatorLogs(ctx, cfg, t)
			t.Fatal(err)
		}
		t.Logf("ClusteRole %s is deleted", InstanaOperatorOldClusterRoleName)
		return ctx
	}
}

func EnsureOldClusterRoleBindingIsGone() e2etypes.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Logf("Ensuring the old clusterrolebinding %s is not running", InstanaOperatorOldClusterRoleBindingName)
		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
		}
		clusterrolebinding := rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: InstanaOperatorOldClusterRoleBindingName},
		}
		err = wait.For(conditions.New(client.Resources()).ResourceDeleted(&clusterrolebinding), wait.WithTimeout(time.Minute*2))
		if err != nil {
			PrintOperatorLogs(ctx, cfg, t)
			t.Fatal(err)
		}
		t.Logf("ClusteRoleBinding %s is deleted", InstanaOperatorOldClusterRoleBindingName)
		return ctx
	}
}

func WaitForAgentSuccessfulBackendConnection() e2etypes.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Log("Searching for successful backend connection in agent logs")
		clientSet, err := kubernetes.NewForConfig(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(20 * time.Second)
		podList, err := clientSet.CoreV1().Pods(cfg.Namespace()).List(ctx, metav1.ListOptions{LabelSelector: "app.kubernetes.io/component=instana-agent"})
		if err != nil {
			t.Fatal(err)
		}
		if len(podList.Items) == 0 {
			t.Fatal("No pods found")
		}

		connectionSuccessful := false
		var buf *bytes.Buffer
		for i := 0; i < 9; i++ {
			t.Log("Sleeping 20 seconds")
			time.Sleep(20 * time.Second)
			t.Log("Fetching logs")
			logReq := clientSet.CoreV1().Pods(cfg.Namespace()).GetLogs(podList.Items[0].Name, &corev1.PodLogOptions{})
			podLogs, err := logReq.Stream(ctx)
			if err != nil {
				t.Fatal("Could not stream logs", err)
			}
			defer podLogs.Close()

			buf = new(bytes.Buffer)
			_, err = io.Copy(buf, podLogs)

			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(buf.String(), "Connected using HTTP/2 to") {
				t.Log("Connection established correctly")
				connectionSuccessful = true
				break
			} else {
				t.Log("Could not find working connection in log of the first pod yet")
			}
		}
		if !connectionSuccessful {
			t.Fatal("Agent pod did not log successful connection, dumping log", buf.String())
		}
		return ctx
	}
}

func ValidateAgentMultiBackendConfiguration() e2etypes.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		log.Infof("Fetching secret %s", InstanaAgentConfigSecretName)
		// Create a client to interact with the Kube API
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}

		// Check if namespace exist, otherwise just skip over it
		instanaAgentConfigSecret := &corev1.Secret{}
		err = r.Get(ctx, InstanaAgentConfigSecretName, InstanaNamespace, instanaAgentConfigSecret)
		if err != nil {
			t.Fatal("Secret could not be fetched", InstanaAgentConfigSecretName, err)
		}

		firstBackendConfigString := string(instanaAgentConfigSecret.Data["com.instana.agent.main.sender.Backend-1.cfg"])
		expectedFirstBackendConfigString := "host=first-backend.instana.io\nport=443\nprotocol=HTTP/2\nkey=xxx\n"
		secondBackendConfigString := string(instanaAgentConfigSecret.Data["com.instana.agent.main.sender.Backend-2.cfg"])
		expectedSecondBackendConfigString := "host=second-backend.instana.io\nport=443\nprotocol=HTTP/2\nkey=yyy\n"

		if firstBackendConfigString != expectedFirstBackendConfigString {
			t.Error("First backend does not match the expected string", firstBackendConfigString, expectedFirstBackendConfigString)
		} else {
			t.Log("First backend config confirmed")
		}
		if secondBackendConfigString != expectedSecondBackendConfigString {
			t.Error("Second backend does not match the expected string", secondBackendConfigString, expectedSecondBackendConfigString)
		} else {
			t.Log("Second backend config confirmed")
		}

		pods := &corev1.PodList{}
		listOps := resources.WithLabelSelector("app.kubernetes.io/component=instana-agent")
		err = r.List(ctx, pods, listOps)
		if err != nil || pods.Items == nil {
			t.Error("error while getting pods", err)
		}
		var stdout, stderr bytes.Buffer
		podName := pods.Items[0].Name
		containerName := "instana-agent"

		backendCheckMatrix := []struct {
			fileSuffix            string
			expectedBackendString string
		}{
			{
				fileSuffix:            "1",
				expectedBackendString: "first-backend.instana.io",
			},
			{
				fileSuffix:            "2",
				expectedBackendString: "second-backend.instana.io",
			},
		}

		for _, currentBackend := range backendCheckMatrix {
			if err := r.ExecInPod(
				ctx,
				cfg.Namespace(),
				podName,
				containerName,
				[]string{"cat", fmt.Sprintf("/opt/instana/agent/etc/instana/com.instana.agent.main.sender.Backend-%s.cfg", currentBackend.fileSuffix)},
				&stdout,
				&stderr,
			); err != nil {
				t.Log(stderr.String())
				t.Error(err)
			}
			if strings.Contains(stdout.String(), currentBackend.expectedBackendString) {
				t.Logf("ExecInPod returned expected backend config for file /opt/instana/agent/etc/instana/com.instana.agent.main.sender.Backend-%s.cfg", currentBackend.fileSuffix)
			} else {
				t.Error(fmt.Sprintf("Expected to find %s in file /opt/instana/agent/etc/instana/com.instana.agent.main.sender.Backend-%s.cfg", currentBackend.expectedBackendString, currentBackend.fileSuffix), stdout.String())
			}
		}

		return ctx
	}
}

func ValidateSecretsMountedFromExtraVolume() e2etypes.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		log.Infof("Fetching secret %s", InstanaAgentConfigSecretName)
		// Create a client to interact with the Kube API
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}

		// Check if namespace exist, otherwise just skip over it
		instanaAgentConfigSecret := &corev1.Secret{}
		err = r.Get(ctx, InstanaAgentConfigSecretName, InstanaNamespace, instanaAgentConfigSecret)
		if err != nil {
			t.Fatal("Secret could not be fetched", InstanaAgentConfigSecretName, err)
		}

		pods := &corev1.PodList{}
		listOps := resources.WithLabelSelector("app.kubernetes.io/component=instana-agent")
		err = r.List(ctx, pods, listOps)
		if err != nil || pods.Items == nil {
			t.Error("error while getting pods", err)
		}
		var stdout, stderr bytes.Buffer
		podName := pods.Items[0].Name
		containerName := "instana-agent"

		secretFileMatrix := []struct {
			path    string
			content string
		}{
			{
				path:    "/secrets/key",
				content: "xxx",
			},
			{
				path:    "/secrets/key-1",
				content: "yyy",
			},
		}

		for _, currentFile := range secretFileMatrix {
			if err := r.ExecInPod(
				ctx,
				cfg.Namespace(),
				podName,
				containerName,
				[]string{"cat", currentFile.path},
				&stdout,
				&stderr,
			); err != nil {
				t.Log(stderr.String())
				t.Error(err)
			}
			if strings.Contains(stdout.String(), "xxx") {
				t.Logf("ExecInPod returned expected secret value from file %s", currentFile.path)
			} else {
				t.Error(fmt.Sprintf("Expected to find %s in file %s", currentFile.content, currentFile.path), stdout.String())
			}
		}

		return ctx
	}
}

// Helper to produce test structs
func NewAgentCr() v1.InstanaAgent {
	boolTrue := true

	return v1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "instana-agent",
			Namespace: InstanaNamespace,
		},
		Spec: v1.InstanaAgentSpec{
			Zone: v1.Name{
				Name: "e2e",
			},
			// ensure to not overlap between concurrent test runs on different clusters, randomize cluster name, but have consistent zone
			Cluster: v1.Name{Name: envconf.RandomName("e2e", 9)},
			Agent: v1.BaseAgentSpec{
				Key:          InstanaTestCfg.InstanaBackend.AgentKey,
				EndpointHost: InstanaTestCfg.InstanaBackend.EndpointHost,
				EndpointPort: strconv.Itoa(InstanaTestCfg.InstanaBackend.EndpointPort),
			},
			OpenTelemetry: v1.OpenTelemetry{
				GRPC: &v1.Enabled{Enabled: &boolTrue},
				HTTP: &v1.Enabled{Enabled: &boolTrue},
			},
		},
	}
}

func PrintOperatorLogs(ctx context.Context, cfg *envconf.Config, t *testing.T) {
	p := utils.RunCommand(
		fmt.Sprintf("kubectl logs deployment/instana-agent-controller-manager -n %s", cfg.Namespace()),
	)
	t.Log("Error while creating pull secret", p.Out())
}
