/*
 * (c) Copyright IBM Corp. 2025
 */

package e2e

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
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

// Constants for namespace names used in selective monitoring tests
const (
	NamespaceNoLabel = "selective-monitoring-no-label"
	NamespaceOptOut  = "selective-monitoring-opt-out"
	NamespaceOptIn   = "selective-monitoring-opt-in"
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

	// Use a static agent image to have faster startup times
	agent.Spec.Agent.ImageSpec.Name = InstanaAgentStaticImage

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
			{NamespaceNoLabel, false, ""},
			{NamespaceOptOut, true, "false"},
			{NamespaceOptIn, true, "true"},
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

		// Get Kubernetes client
		clientSet, err := kubernetes.NewForConfig(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}

		// Define the namespaces where our demo apps are running
		namespaces := []string{
			NamespaceNoLabel,
			NamespaceOptOut,
			NamespaceOptIn,
		}

		// Find all demo app pods across all namespaces
		t.Log("Finding all demo app pods...")
		var demoAppPods []corev1.Pod
		var demoAppNodes = make(map[string]bool)
		var podsByNamespace = make(map[string][]corev1.Pod)

		for _, ns := range namespaces {
			podList, err := clientSet.CoreV1().Pods(ns).List(
				ctx,
				metav1.ListOptions{
					LabelSelector: "app=java-demo-app,e2etest=selective-monitoring",
				},
			)
			if err != nil {
				t.Logf("Error listing pods in namespace %s: %v", ns, err)
				continue
			}

			podsByNamespace[ns] = podList.Items
			for _, pod := range podList.Items {
				demoAppPods = append(demoAppPods, pod)
				demoAppNodes[pod.Spec.NodeName] = true
				t.Logf("Found demo app pod %s in namespace %s on node %s",
					pod.Name, pod.Namespace, pod.Spec.NodeName)
			}
		}

		if len(demoAppPods) == 0 {
			t.Fatal("No demo app pods found in any namespace")
		}

		// Get all agent pods
		agentPodList, err := clientSet.CoreV1().Pods(cfg.Namespace()).List(
			ctx,
			metav1.ListOptions{LabelSelector: "app.kubernetes.io/component=instana-agent"},
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(agentPodList.Items) == 0 {
			t.Fatal("No agent pods found")
		}

		// Find agent pods running on the same nodes as our demo apps
		var relevantAgentPods []corev1.Pod
		for _, agentPod := range agentPodList.Items {
			if demoAppNodes[agentPod.Spec.NodeName] {
				relevantAgentPods = append(relevantAgentPods, agentPod)
				t.Logf("Found agent pod %s on node %s that has demo app pods",
					agentPod.Name, agentPod.Spec.NodeName)
			}
		}

		if len(relevantAgentPods) == 0 {
			t.Fatal("No agent pods found on nodes where demo apps are running")
		}

		// Define regex patterns for finding PIDs and attachment logs
		vmDiscoveryRegex := regexp.MustCompile(
			`adding new VM with PID (\d+).*INSTANA_TEST_POD_NAME=([^,\s]+)`,
		)

		// Track attachment status for each namespace
		attachmentStatus := map[string]bool{
			NamespaceNoLabel: false,
			NamespaceOptOut:  false,
			NamespaceOptIn:   false,
		}

		// Set up polling parameters
		pollInterval := 10 * time.Second
		maxPollTime := 5 * time.Minute
		startTime := time.Now()
		deadline := startTime.Add(maxPollTime)

		// Poll until we find the expected attachment pattern or timeout
		t.Log("Starting to poll agent logs for JVM attachment (up to 5 minutes)...")

		for time.Now().Before(deadline) {
			// Check if we've already found attachments that shouldn't happen
			if attachmentStatus[NamespaceNoLabel] ||
				attachmentStatus[NamespaceOptOut] {
				t.Error("Found JVM attachment in namespace that should not be monitored")
				break
			}

			// Check if we've found the expected attachment
			if attachmentStatus[NamespaceOptIn] {
				t.Log("Found expected JVM attachment in opt-in namespace")
				break
			}

			// Check logs from all relevant agent pods
			for _, agentPod := range relevantAgentPods {
				t.Logf("Checking logs from agent pod %s on node %s",
					agentPod.Name, agentPod.Spec.NodeName)

				var buf bytes.Buffer
				logReq := clientSet.CoreV1().
					Pods(cfg.Namespace()).
					GetLogs(agentPod.Name, &corev1.PodLogOptions{})
				podLogs, err := logReq.Stream(ctx)
				if err != nil {
					t.Logf("Could not stream logs from pod %s: %v", agentPod.Name, err)
					continue
				}

				_, copyErr := io.Copy(&buf, podLogs)
				if err := podLogs.Close(); err != nil {
					t.Logf("Error closing log stream for pod %s: %v", agentPod.Name, err)
				}
				if copyErr != nil {
					t.Logf("Error reading logs from pod %s: %v", agentPod.Name, err)
					continue
				}

				logs := buf.String()
				t.Logf("Agent logs retrieved from pod %s, checking for JVM attachment...",
					agentPod.Name)

				// For each namespace, check if its pods are being monitored
				for ns, pods := range podsByNamespace {
					for _, pod := range pods {
						podName := pod.Name

						// Look for VM discovery log entries for this pod
						matches := vmDiscoveryRegex.FindAllStringSubmatch(logs, -1)
						for _, match := range matches {
							if len(match) >= 3 && strings.Contains(match[2], podName) {
								// Found a VM discovery log for this pod
								pid := match[1]
								t.Logf("Found VM discovery for pod %s in namespace %s with PID %s",
									podName, ns, pid)

								// Check if there's an initial attach attempt
								attachAttemptPattern := fmt.Sprintf(
									"Performing initial attach to JVM with PID %s",
									pid,
								)
								if strings.Contains(logs, attachAttemptPattern) {
									t.Logf(
										"Found attach attempt for pod %s (PID %s) in namespace %s",
										podName,
										pid,
										ns,
									)

									// Check if the attachment was successful
									attachSuccessPattern := fmt.Sprintf(
										"Initial attach to JVM with PID %s successful",
										pid,
									)
									if strings.Contains(logs, attachSuccessPattern) {
										t.Logf(
											"Found successful attachment for pod %s (PID %s) in namespace %s",
											podName,
											pid,
											ns,
										)
										attachmentStatus[ns] = true
									}
								}
							}
						}
					}
				}
			}

			// If we haven't found what we're looking for, wait before checking again
			if !attachmentStatus[NamespaceOptIn] &&
				!attachmentStatus[NamespaceNoLabel] &&
				!attachmentStatus[NamespaceOptOut] {
				t.Logf("No definitive attachment status found yet, polling again in %v...",
					pollInterval)
				time.Sleep(pollInterval)
			} else {
				break
			}
		}

		// Log polling duration
		pollDuration := time.Since(startTime)
		t.Logf("Finished polling after %v", pollDuration)

		// Verify expectations
		if !attachmentStatus[NamespaceOptIn] {
			t.Error(
				"JVM in opt-in namespace should be monitored, but no attachment was found in logs",
			)
		} else {
			t.Log("JVM in opt-in namespace is correctly monitored")
		}

		if attachmentStatus[NamespaceNoLabel] {
			t.Error(
				"JVM in no-label namespace should not be monitored, but attachment was found in logs",
			)
		} else {
			t.Log("JVM in no-label namespace is correctly not monitored")
		}

		if attachmentStatus[NamespaceOptOut] {
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
			NamespaceNoLabel,
			NamespaceOptOut,
			NamespaceOptIn,
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
