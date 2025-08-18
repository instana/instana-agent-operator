/*
 * (c) Copyright IBM Corp. 2025
 */

package e2e

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	v1 "github.com/instana/instana-agent-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/support/utils"
)

// TestSelectiveMonitoring tests the selective monitoring feature of the Instana agent operator.
// It deploys the agent in opt-in mode and verifies that only JVMs in namespaces with the
// instana-workload-monitoring=true label are monitored.
func TestSelectiveMonitoring(t *testing.T) {
	// Create a new agent CR with selective monitoring enabled
	agent := NewAgentCrWithSelectiveMonitoring()

	// Define the test feature
	selectiveMonitoringFeature := features.New("selective monitoring in opt-in mode").
		Setup(SetupOperatorDevBuild()).
		Setup(WaitForDeploymentToBecomeReady(InstanaOperatorDeploymentName)).
		Setup(DeployAgentCr(&agent)).
		Setup(DeployJavaDemoAppInNamespaces()).
		Assess("wait for k8sensor deployment to become ready", WaitForDeploymentToBecomeReady(K8sensorDeploymentName)).
		Assess("wait for agent daemonset to become ready", WaitForAgentDaemonSetToBecomeReady()).
		Assess("verify selective monitoring works correctly", VerifySelectiveMonitoring()).
		Teardown(CleanupNamespaces()).
		Feature()

	// Run the test
	testEnv.Test(t, selectiveMonitoringFeature)
}

// NewAgentCrWithSelectiveMonitoring creates a new agent CR with selective monitoring enabled in opt-in mode.
func NewAgentCrWithSelectiveMonitoring() v1.InstanaAgent {
	agent := NewAgentCr() // Use the existing function to create a base agent CR

	// Set the INSTANA_SELECTIVE_MONITORING environment variable using the Kubernetes style format
	if agent.Spec.Agent.Pod.Env == nil {
		agent.Spec.Agent.Pod.Env = []corev1.EnvVar{}
	}

	agent.Spec.Agent.Pod.Env = append(agent.Spec.Agent.Pod.Env, corev1.EnvVar{
		Name:  "INSTANA_SELECTIVE_MONITORING",
		Value: "OPT_IN",
	})

	return agent
}

// DeployJavaDemoAppInNamespaces deploys the Java demo app in three different namespaces with appropriate labels.
func DeployJavaDemoAppInNamespaces() features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		// Define the namespaces and their configurations
		namespaces := []struct {
			name     string
			addLabel bool
			value    string
		}{
			{"selective-monitoring-no-label", false, ""},
			{"selective-monitoring-opt-out", true, "false"},
			{"selective-monitoring-opt-in", true, "true"},
		}

		// Use a wait group to deploy all apps concurrently
		var wg sync.WaitGroup
		for _, ns := range namespaces {
			wg.Add(1)
			go func(namespace string, addLabel bool, labelValue string) {
				defer wg.Done()
				deployJavaDemoApp(ctx, t, namespace, addLabel, labelValue)
			}(ns.name, ns.addLabel, ns.value)
		}

		// Wait for all deployments to be created
		wg.Wait()
		t.Log("All demo apps have been deployed, now waiting for them to become ready")

		// Create a client to interact with the Kube API
		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
		}

		// Wait for all deployments to be ready concurrently
		var waitWg sync.WaitGroup
		for _, ns := range namespaces {
			waitWg.Add(1)
			go func(namespace string) {
				defer waitWg.Done()
				deploymentName := "java-demo-app"

				t.Logf(
					"Waiting for deployment %s in namespace %s to become ready",
					deploymentName,
					namespace,
				)

				// Create a deployment reference
				dep := appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{Name: deploymentName, Namespace: namespace},
				}

				// Wait for the deployment to be ready
				err := wait.For(
					conditions.New(client.Resources()).
						DeploymentConditionMatch(&dep, appsv1.DeploymentAvailable, corev1.ConditionTrue),
					wait.WithTimeout(time.Minute*2),
				)
				if err != nil {
					t.Logf(
						"Error waiting for deployment %s in namespace %s: %v",
						deploymentName,
						namespace,
						err,
					)
					return
				}

				t.Logf("Deployment %s in namespace %s is ready", deploymentName, namespace)
			}(ns.name)
		}

		// Wait for all deployments to be ready
		waitWg.Wait()
		t.Log("All demo app deployments are ready")

		return ctx
	}
}

// deployJavaDemoApp deploys the Java demo app in the specified namespace with the specified label.
func deployJavaDemoApp(
	ctx context.Context,
	t *testing.T,
	namespace string,
	addLabel bool,
	labelValue string,
) {
	// Delete the namespace if it exists and wait for it to be fully deleted
	t.Logf("Ensuring clean namespace %s...", namespace)
	p := utils.RunCommand(
		fmt.Sprintf("kubectl delete ns %s --ignore-not-found --wait --timeout=30s", namespace),
	)
	if p.Err() != nil {
		t.Logf("Error deleting namespace %s: %v", namespace, p.Err())
	}

	// Create namespace
	t.Logf("Creating namespace %s...", namespace)
	p = utils.RunCommand(fmt.Sprintf("kubectl create ns %s", namespace))
	if p.Err() != nil {
		t.Fatal("Error creating namespace:", p.Err())
	}

	// Add label if needed
	if addLabel {
		p = utils.RunCommand(
			fmt.Sprintf(
				"kubectl label ns %s instana-workload-monitoring=%s",
				namespace,
				labelValue,
			),
		)
		if p.Err() != nil {
			t.Fatal("Error labeling namespace:", p.Err())
		}
	}

	// Check if the registry configuration exists
	if InstanaTestCfg.ContainerRegistry == nil {
		t.Fatal("Container registry configuration is not set in the test configuration")
	}

	// Secret name - use the same name as in SetupOperatorDevBuild()
	secretName := InstanaTestCfg.ContainerRegistry.Name

	// Check if secret exists in the target namespace
	t.Logf("Checking if pull secret exists in namespace %s...", namespace)
	p = utils.RunCommand(fmt.Sprintf("kubectl get secret %s -n %s", secretName, namespace))
	secretExists := p.Err() == nil

	// Delete secret if it exists
	if secretExists {
		t.Logf("Updating existing pull secret...")
		p = utils.RunCommand(fmt.Sprintf("kubectl delete secret %s -n %s", secretName, namespace))
		if p.Err() != nil {
			t.Fatal("Error deleting existing pull secret:", p.Err())
		}
	} else {
		t.Logf("Creating new pull secret...")
	}

	// Create docker-registry secret directly using kubectl with the same config values as in SetupOperatorDevBuild()
	p = utils.RunCommand(fmt.Sprintf(
		"kubectl create secret docker-registry %s --docker-server=%s --docker-username=%s --docker-password=%s -n %s",
		InstanaTestCfg.ContainerRegistry.Name,
		InstanaTestCfg.ContainerRegistry.Host,
		InstanaTestCfg.ContainerRegistry.User,
		InstanaTestCfg.ContainerRegistry.Password,
		namespace,
	))
	if p.Err() != nil {
		t.Fatal("Error creating pull secret:", p.Err())
	}

	// Apply deployment
	deploymentPath := "java-demo-app/deployment.yaml"
	t.Logf("Applying deployment from path: %s to namespace: %s", deploymentPath, namespace)

	// Apply the deployment with the correct path
	applyCmd := fmt.Sprintf("kubectl apply -f %s -n %s", deploymentPath, namespace)
	t.Logf("Applying deployment with command: %s", applyCmd)
	p = utils.RunCommand(applyCmd)
	if p.Err() != nil {
		t.Fatalf(
			"Error applying deployment from %s to namespace %s: %v",
			deploymentPath,
			namespace,
			p.Err(),
		)
	}
	t.Logf("Successfully applied deployment to namespace %s", namespace)
}

// VerifySelectiveMonitoring verifies that only the JVM in the opt-in namespace is monitored.
func VerifySelectiveMonitoring() features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Log("Verifying selective monitoring...")
		// TODO: Needs redefinement, the agent pod must run on the same worker node as the demo app to be monitored
		// First notes: Selective monitoring is set to OPT_IN in the agent log
		// 2025-08-18T14:04:59.606+00:00 | INFO  | instana-executor-thread-2-3      | lMachineUtilImpl | com.instana.agent-util - 1.0.12 | addToVmMapWithLogAndRequestDiscovery | adding new VM with PID 3537885, ContainerizedVirtualMachineImpl [pid=3537885, inContainerPid=1, containerFileSystem=/proc/3537885/root, parentPid=3537885, attachType=HOTSPOT_MODULAR, hasAttachFile=false, commandLine=HelloWorldServer, vmArgs=, name=OpenJDK 64-Bit Server VM (Red_Hat-21.0.8.0.9-1), vendor=OpenJDK, version=21.0.8, build=21.0.8+9-LTS, heapCapacity=-1, process=CrioProcessImpl [pid=3537885, parentPid=3537883, inContainerPid=1, containerId=3d9cc71389d96e80221057c69022b13eaa74b7dbb5e657846573086c0363b424, containerFileSystem=/proc/3537885/root, startTime=0, name=java, gatewayAddress='<lazy>, anyListenAddress='<lazy>, directory=/app, executable=/usr/lib/jvm/java-21-openjdk-21.0.8.0.9-1.el9.x86_64/bin/java, arguments=[HelloWorldServer], userName=<lazy>, groupName=<lazy>, procCred=null, environmentVariables={PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin, KUBERNETES_PORT_443_TCP=tcp://172.30.0.1:443, container=oci, KUBERNETES_PORT_443_TCP_ADDR=172.30.0.1, KUBERNETES_PORT=tcp://172.30.0.1:443, KUBERNETES_PORT_443_TCP_PROTO=tcp, TERM=xterm, USER_UID=1001, KUBERNETES_SERVICE_HOST=172.30.0.1, KUBERNETES_SERVICE_PORT=443, HOSTNAME=java-demo-app-79bcdf6476-wdtff, NSS_SDB_USE_CACHE=no, KUBERNETES_PORT_443_TCP_PORT=443, KUBERNETES_SERVICE_PORT_HTTPS=443, HOME=/}, stable=true]] //nolint:lll
		// Grep for the agent logs:
		// "HelloWorldServer" and "adding new VM with PID xxx". Isolate the used PID which is required for the next request.
		// With the PID, check if the agent log contains a successful attachment for it.
		// The log would be "Initial attach to JVM with PID xxx successful".

		// Get agent pods
		clientSet, err := kubernetes.NewForConfig(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}

		// Wait for the agent to have time to discover and attach to JVMs
		t.Log("Waiting for agent to discover and attach to JVMs...")
		time.Sleep(60 * time.Second)

		podList, err := clientSet.CoreV1().Pods(cfg.Namespace()).List(
			ctx,
			metav1.ListOptions{LabelSelector: "app.kubernetes.io/component=instana-agent"},
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(podList.Items) == 0 {
			t.Fatal("No agent pods found")
		}

		// Check logs for JVM attachment
		var buf bytes.Buffer
		logReq := clientSet.CoreV1().
			Pods(cfg.Namespace()).
			GetLogs(podList.Items[0].Name, &corev1.PodLogOptions{})
		podLogs, err := logReq.Stream(ctx)
		if err != nil {
			t.Fatal("Could not stream logs", err)
		}
		defer podLogs.Close()

		_, err = io.Copy(&buf, podLogs)
		if err != nil {
			t.Fatal(err)
		}

		logs := buf.String()
		t.Logf("Agent logs retrieved, checking for JVM attachment...")

		// Check for successful JVM attachment in the opt-in namespace
		optInAttached := strings.Contains(logs, "Initial attach to JVM") &&
			strings.Contains(logs, "successful") &&
			strings.Contains(logs, "selective-monitoring-opt-in")

		// Check for absence of JVM attachment in the other namespaces
		noLabelAttached := strings.Contains(logs, "Initial attach to JVM") &&
			strings.Contains(logs, "successful") &&
			strings.Contains(logs, "selective-monitoring-no-label")

		optOutAttached := strings.Contains(logs, "Initial attach to JVM") &&
			strings.Contains(logs, "successful") &&
			strings.Contains(logs, "selective-monitoring-opt-out")

		// Verify expectations
		if !optInAttached {
			t.Error(
				"JVM in opt-in namespace should be monitored, but no attachment was found in logs",
			)
		} else {
			t.Log("JVM in opt-in namespace is correctly monitored")
		}

		if noLabelAttached {
			t.Error(
				"JVM in no-label namespace should not be monitored, but attachment was found in logs",
			)
		} else {
			t.Log("JVM in no-label namespace is correctly not monitored")
		}

		if optOutAttached {
			t.Error(
				"JVM in opt-out namespace should not be monitored, but attachment was found in logs",
			)
		} else {
			t.Log("JVM in opt-out namespace is correctly not monitored")
		}

		return ctx
	}
}

// CleanupNamespaces cleans up the namespaces created for the test.
func CleanupNamespaces() features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		namespaces := []string{
			"selective-monitoring-no-label",
			"selective-monitoring-opt-out",
			"selective-monitoring-opt-in",
		}

		for _, ns := range namespaces {
			p := utils.RunCommand(fmt.Sprintf("kubectl delete ns %s --wait --timeout=30s", ns))
			if p.Err() != nil {
				t.Logf("Error deleting namespace %s: %v", ns, p.Err())
			} else {
				t.Logf("Namespace %s deleted successfully", ns)
			}
		}

		return ctx
	}
}

// Made with Bob
