/*
 * (c) Copyright IBM Corp. 2025
 * (c) Copyright Instana Inc. 2025
 */

package e2e

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	v1 "github.com/instana/instana-agent-operator/api/v1"
	securityv1 "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1"
	corev1 "k8s.io/api/core/v1"
	log "k8s.io/klog/v2"

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

var (
	AgentRemoteDeploymentName     = "instana-agent-r-"
	AgentRemoteCustomResourceName = "remote-agent"
)

func NewAgentRemoteCr(name string) v1.InstanaAgentRemote {

	return v1.InstanaAgentRemote{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: InstanaNamespace,
		},
		Spec: v1.InstanaAgentRemoteSpec{
			Zone: v1.Name{
				Name: "e2e",
			},
			ConfigurationYaml: "testing",
		},
	}
}

func DeployAgentRemoteCr(agent *v1.InstanaAgentRemote) e2etypes.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
		}
		log.Info("Creating a new Agent Remote CR")

		// Create Agent Remote CR
		r := client.Resources(cfg.Namespace())
		err = v1.AddToScheme(r.GetScheme())
		if err != nil {
			log.Fatal("Could not add Agent Remote CR to client scheme", err)
		}

		err = r.Create(ctx, agent)
		if err != nil {
			log.Fatal("Could not create Agent Remote CR", err)
		}

		return ctx
	}
}

func WaitForAgentRemoteSuccessfulBackendConnection() e2etypes.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		log.Info("Searching for successful backend connection in agent remote logs")
		clientSet, err := kubernetes.NewForConfig(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(20 * time.Second)
		podList, err := clientSet.CoreV1().Pods(cfg.Namespace()).List(ctx, metav1.ListOptions{LabelSelector: "app.kubernetes.io/component=instana-agent-remote"})
		if err != nil {
			log.Fatal(err)
		}
		if len(podList.Items) == 0 {
			log.Fatal("No pods found")
		}

		connectionSuccessful := false
		var buf *bytes.Buffer
		for i := 0; i < 9; i++ {
			log.Info("Sleeping 20 seconds")
			time.Sleep(20 * time.Second)
			log.Info("Fetching logs")
			logReq := clientSet.CoreV1().Pods(cfg.Namespace()).GetLogs(podList.Items[0].Name, &corev1.PodLogOptions{})
			podLogs, err := logReq.Stream(ctx)
			if err != nil {
				log.Fatal("Could not stream logs", err)
			}
			defer podLogs.Close()

			buf = new(bytes.Buffer)
			_, err = io.Copy(buf, podLogs)

			if err != nil {
				log.Fatal(err)
			}
			if strings.Contains(buf.String(), "Connected using HTTP/2 to") {
				log.Info("Connection established correctly")
				connectionSuccessful = true
				break
			} else {
				log.Info("Could not find working connection in log of the first pod yet")
			}
		}
		if !connectionSuccessful {
			log.Fatal("Agent pod did not log successful connection, dumping log", buf.String())
		}
		return ctx
	}
}

func EnsureAgentRemoteDeletion() env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		log.Info("==== Starting Cleanup, errors are expected if resources are not available ====")
		log.Infof("Ensure namespace %s is not present", cfg.Namespace())

		// Create a client to interact with the Kube API
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			return ctx, fmt.Errorf("failed to initialize client: %v", err)
		}

		p := utils.RunCommand("kubectl get pods -n instana-agent")
		log.Info("Current pods: ", p.Command(), p.ExitCode(), "\n", p.Result())

		p = utils.RunCommand("kubectl get agentremote remote-agent -o yaml -n instana-agent")
		// redact agent key if present
		log.Info("Current agent remote CR: ", p.Command(), p.ExitCode(), "\n", strings.ReplaceAll(p.Result(), InstanaTestCfg.InstanaBackend.AgentKey, "***"))

		// Cleanup a potentially existing Agent CR first
		if _, err := DeleteAgentRemoteCRIfPresent()(ctx, cfg); err != nil {
			log.Info("Agent CR cleanup err: ", err)
		}

		log.Info("Agent CR cleanup completed")

		// full purge of remote resources if anything would be left in the cluster
		// Removing the finalizer from the existing Agent CR to make it deletable
		// kubectl patch agent instana-agent-remote -p '{"metadata":{"finalizers":[]}}' --type=merge
		agent := &v1.InstanaAgentRemote{}
		log.Info("Patching agent remote cr to remove finalizers")
		err = r.Patch(ctx, agent, k8s.Patch{
			PatchType: types.MergePatchType,
			Data:      []byte(`{"metadata":{"finalizers":[]}}`),
		})
		if err != nil {
			return ctx, fmt.Errorf("cleanup: Patch agent remote CR failed: %v", err)
		}

		p = utils.RunCommand("kubectl delete crd/agentsremote.instana.io")
		if p.Err() != nil {
			log.Warningf("Could not remove some artifacts, ignoring as they might not be present %s - %s - %s - %d", p.Command(), p.Err(), p.Out(), p.ExitCode())
		}

		log.Info("==== Cleanup compleated ====")
		return ctx, nil
	}
}

// On OpenShift we need to ensure the instana-agent service account gets permission to the privilged security context
// This action is only necessary once per OCP cluster as it is not tight to a namespace, but to a cluster
func AdjustOcpPermissionsIfNecessaryRemote() env.Func {
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
			command := "oc adm policy add-scc-to-user anyuid -z instana-agent-remote -n instana-agent"
			log.Infof("OpenShift detected, adding instana-agent-remote service account to SecurityContextConstraints via api, command would be: %s\n", command)

			// replaced command execution with SDK call to not require `oc` cli
			securityClient, err := securityv1.NewForConfig(cfg.Client().RESTConfig())
			if err != nil {
				return ctx, fmt.Errorf("could not initialize securityClient: %v", err)
			}

			// get security context constraints
			scc, err := securityClient.SecurityContextConstraints().Get(ctx, "anyuid", metav1.GetOptions{})
			if err != nil {
				return ctx, fmt.Errorf("failed to get SecurityContextContraints: %v", err)
			}

			// check if service account user for remote agent is already listed in the scc
			serviceAccountId := fmt.Sprintf("system:serviceaccount:%s:%s", InstanaNamespace, "instana-agent-remote")
			userFound := false

			for _, user := range scc.Users {
				if user == serviceAccountId {
					userFound = true
					break
				}
			}

			if userFound {
				log.Infof("Security Context Constraint \"anyuid\" already lists service account user: %v\n", serviceAccountId)
				return ctx, nil
			}

			// updating Security Context Constraints to list instana remote service account
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

func DeleteAgentRemoteCRIfPresent() env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		log.Info("Ensure agent remote CR is not present")
		// Create a client to interact with the Kube API
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			return ctx, fmt.Errorf("cleanup: Error initializing client to delete agent remote CR: %v", err)
		}

		// Assume an existing namespace at this point, check if an agent remote CR is present (requires to adjust schema of current client)
		r.WithNamespace(InstanaNamespace)
		err = v1.AddToScheme(r.GetScheme())
		if err != nil {
			// If this fails, the cleanup will not work properly -> failing
			return ctx, fmt.Errorf("cleanup: Error could not add agent remote types to current scheme: %v", err)
		}

		// If the agent remote cr is available, but the operator is already gone, the finalizer will never be removed
		// This will lead to a delayed namespace termination which never completes. To avoid that, patch the agent remote CR
		// to remove the finalizer. Afterwards, it can be successfully deleted.
		agent := &v1.InstanaAgentRemote{}
		err = r.Get(ctx, AgentRemoteCustomResourceName, InstanaNamespace, agent)
		if errors.IsNotFound(err) {
			// No agent cr found, skip this cleanup step
			log.Info("No agent remote CR present, skipping deletion")
			return ctx, nil
		}

		// The agent CR could not be fetched due to a different reason, failing
		if err != nil {
			return ctx, fmt.Errorf("cleanup: Fetch agent remote CR failed: %v", err)
		}

		// Removing the finalizer from the existing Agent CR to make it deletable
		// kubectl patch agent instana-agent-remote -p '{"metadata":{"finalizers":[]}}' --type=merge
		log.Info("Patching agent remote cr to remove finalizers")
		err = r.Patch(ctx, agent, k8s.Patch{
			PatchType: types.MergePatchType,
			Data:      []byte(`{"metadata":{"finalizers":[]}}`),
		})
		if err != nil {
			return ctx, fmt.Errorf("cleanup: Patch agent remote CR failed: %v", err)
		}

		log.Info("Deleting CR")
		// delete explicitly, namespace deletion would delete the agent CR as well if the finalizer is not present
		err = r.Delete(ctx, agent)

		if err != nil {
			// The deletion failed for some reason, failing the cleanup
			return ctx, fmt.Errorf("cleanup: Delete agent remote CR failed: %v", err)
		}

		agentCrList := &v1.InstanaAgentRemoteList{
			Items: []v1.InstanaAgentRemote{*agent},
		}

		// Ensure to wait for the agent CR to disappear before continuing
		err = wait.For(conditions.New(r).ResourcesDeleted(agentCrList))
		if err != nil {
			return ctx, fmt.Errorf("cleanup: Waiting for agent remote CR deletion failed: %v", err)
		}
		log.Info("Agent Remote CR is gone")
		return ctx, nil
	}
}
