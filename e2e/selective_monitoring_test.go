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
	"testing"
	"time"

	v1 "github.com/instana/instana-agent-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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
		// Deploy in namespace without label
		deployJavaDemoApp(ctx, t, "selective-monitoring-no-label", false, "")

		// Deploy in namespace with opt-out label
		deployJavaDemoApp(ctx, t, "selective-monitoring-opt-out", true, "false")

		// Deploy in namespace with opt-in label
		deployJavaDemoApp(ctx, t, "selective-monitoring-opt-in", true, "true")

		// Wait for all deployments to be ready
		time.Sleep(30 * time.Second)

		return ctx
	}
}

// deployJavaDemoApp deploys the Java demo app in the specified namespace with the specified label.
func deployJavaDemoApp(ctx context.Context, t *testing.T, namespace string, addLabel bool, labelValue string) {
	// Create namespace
	p := utils.RunCommand(fmt.Sprintf("kubectl create ns %s", namespace))
	if p.Err() != nil {
		t.Logf("Namespace %s might already exist: %v", namespace, p.Err())
	}

	// Add label if needed
	if addLabel {
		p = utils.RunCommand(fmt.Sprintf("kubectl label ns %s instana-workload-monitoring=%s", namespace, labelValue))
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
		namespace))
	if p.Err() != nil {
		t.Fatal("Error creating pull secret:", p.Err())
	}

	// Apply deployment
	deploymentPath := "e2e/java-demo-app/deployment.yaml"
	p = utils.RunCommand(fmt.Sprintf("kubectl apply -f %s -n %s", deploymentPath, namespace))
	if p.Err() != nil {
		t.Fatal("Error applying deployment:", p.Err())
	}

	// Wait for deployment to be ready
	p = utils.RunCommand(
		fmt.Sprintf("kubectl wait --for=condition=available e2e/deployment/java-demo-app -n %s --timeout=120s", namespace),
	)
	if p.Err() != nil {
		t.Fatal("Error waiting for Java demo app deployment:", p.Err())
	}

	t.Logf("Java demo app deployed successfully in namespace %s", namespace)
}

// VerifySelectiveMonitoring verifies that only the JVM in the opt-in namespace is monitored.
func VerifySelectiveMonitoring() features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Log("Verifying selective monitoring...")

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
		logReq := clientSet.CoreV1().Pods(cfg.Namespace()).GetLogs(podList.Items[0].Name, &corev1.PodLogOptions{})
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
			t.Error("JVM in opt-in namespace should be monitored, but no attachment was found in logs")
		} else {
			t.Log("JVM in opt-in namespace is correctly monitored")
		}

		if noLabelAttached {
			t.Error("JVM in no-label namespace should not be monitored, but attachment was found in logs")
		} else {
			t.Log("JVM in no-label namespace is correctly not monitored")
		}

		if optOutAttached {
			t.Error("JVM in opt-out namespace should not be monitored, but attachment was found in logs")
		} else {
			t.Log("JVM in opt-out namespace is correctly not monitored")
		}

		return ctx
	}
}

// CleanupNamespaces cleans up the namespaces created for the test.
// This function is not used directly in the test but can be used for manual cleanup.
func CleanupNamespaces() features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		namespaces := []string{
			"selective-monitoring-no-label",
			"selective-monitoring-opt-out",
			"selective-monitoring-opt-in",
		}

		for _, ns := range namespaces {
			p := utils.RunCommand(fmt.Sprintf("kubectl delete ns %s", ns))
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
