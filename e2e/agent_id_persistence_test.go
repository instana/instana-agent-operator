/*
 * (c) Copyright IBM Corp. 2026
 */

package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/pkg/utils"
)

// TestAgentIDPersistence verifies that the agent ID persists across pod restarts
// when INSTANA_PERSIST_HOST_UNIQUE_ID is set and /var/lib is mounted
func TestAgentIDPersistence(t *testing.T) {
	agent := NewAgentCr()

	feature := features.New("agent ID persistence across pod restarts").
		Setup(SetupOperatorDevBuild()).
		Setup(DeployAgentCr(&agent)).
		Assess("wait for instana-agent-controller-manager deployment to become ready",
			WaitForDeploymentToBecomeReady(InstanaOperatorDeploymentName)).
		Assess("wait for k8sensor deployment to become ready",
			WaitForDeploymentToBecomeReady(K8sensorDeploymentName)).
		Assess("wait for agent daemonset to become ready",
			WaitForAgentDaemonSetToBecomeReady()).
		Assess("verify INSTANA_PERSIST_HOST_UNIQUE_ID env var is set",
			verifyPersistHostUniqueIDEnvVar())

	// Skip persistence verification when cloud provider metadata is available
	// Cloud provider IDs take precedence and don't use file persistence
	if !isCloudProviderMetadataAvailable(t) {
		feature = feature.
			Assess("get initial agent ID from pod",
				getAndStoreAgentID()).
			Assess("delete agent pod to trigger restart",
				deleteAgentPod()).
			Assess("wait for agent daemonset to become ready after restart",
				WaitForAgentDaemonSetToBecomeReady()).
			Assess("verify agent ID persisted after restart",
				verifyAgentIDPersisted())
	} else {
		t.Log(
			"Skipping agent ID persistence verification - " +
				"cloud provider metadata available, agent will use cloud provider ID",
		)
	}

	agentIDPersistenceFeature := feature.Feature()

	testEnv.Test(t, agentIDPersistenceFeature)
}

// verifyPersistHostUniqueIDEnvVar checks that the INSTANA_PERSIST_HOST_UNIQUE_ID env var is set
func verifyPersistHostUniqueIDEnvVar() features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Log("Verifying INSTANA_PERSIST_HOST_UNIQUE_ID environment variable is set")

		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal("Failed to create client:", err)
		}

		// Get agent pods
		pods := &corev1.PodList{}
		listOps := resources.WithLabelSelector("app.kubernetes.io/component=instana-agent")
		err = r.List(ctx, pods, listOps)
		if err != nil || len(pods.Items) == 0 {
			t.Fatal("Error while getting agent pods:", err)
		}

		pod := pods.Items[0]

		// Find the instana-agent container
		var agentContainer *corev1.Container
		for i := range pod.Spec.Containers {
			if pod.Spec.Containers[i].Name == "instana-agent" {
				agentContainer = &pod.Spec.Containers[i]
				break
			}
		}

		if agentContainer == nil {
			t.Fatal("instana-agent container not found in pod")
		}

		// Check for the env var
		found := false
		for _, env := range agentContainer.Env {
			if env.Name == "INSTANA_PERSIST_HOST_UNIQUE_ID" {
				found = true
				if env.Value != "true" {
					t.Fatalf(
						"INSTANA_PERSIST_HOST_UNIQUE_ID has unexpected value: %s (expected: true)",
						env.Value,
					)
				}
				t.Log("✓ INSTANA_PERSIST_HOST_UNIQUE_ID is set to 'true'")
				break
			}
		}

		if !found {
			t.Fatal("INSTANA_PERSIST_HOST_UNIQUE_ID environment variable not found")
		}

		return ctx
	}
}

// getAndStoreAgentID retrieves the agent ID from the pod and stores it in context
// It first waits for the agent to log that it has persisted the ID, then reads the file
func getAndStoreAgentID() features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Log("Getting initial agent ID from pod")

		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal("Failed to create client:", err)
		}

		// Get agent pods
		pods := &corev1.PodList{}
		listOps := resources.WithLabelSelector("app.kubernetes.io/component=instana-agent")
		err = r.List(ctx, pods, listOps)
		if err != nil || len(pods.Items) == 0 {
			t.Fatal("Error while getting agent pods:", err)
		}

		pod := pods.Items[0]
		podName := pod.Name
		nodeName := pod.Spec.NodeName
		t.Logf("Waiting for agent to persist ID in pod: %s on node: %s", podName, nodeName)

		// First, wait for the agent to log that it has persisted the ID
		// 6 minute timeout
		maxRetries := 72 // 72 * 5 seconds = 6 minutes
		retryInterval := 5 * time.Second
		logFound := false

		for i := 0; i < maxRetries; i++ {
			// Use kubectl logs to check for the agent ID file path log
			p := utils.RunCommand(
				fmt.Sprintf(
					"bash -c \"kubectl logs pod/%s -n %s -c instana-agent | grep 'Using agent ID file path:' || echo 'not found'\"",
					podName,
					cfg.Namespace(),
				),
			)

			output := strings.TrimSpace(p.Result())
			if p.Err() == nil && output != "not found" && output != "" {
				t.Logf("✓ Agent is using ID file path: %s", output)
				logFound = true
				break
			}

			if i == 0 {
				t.Log("Waiting for agent to persist ID (checking logs)...")
			}

			if i < maxRetries-1 {
				time.Sleep(retryInterval)
			}
		}

		if !logFound {
			t.Fatalf(
				"Agent did not log 'Using agent ID file path:' after %d minutes",
				maxRetries*int(retryInterval.Seconds())/60,
			)
		}

		// Now read the agent ID file using kubectl exec
		t.Logf("Reading agent ID file from pod: %s", podName)
		var agentID string

		for i := 0; i < 5; i++ { // Shorter retry since we already waited for the log
			p := utils.RunCommand(
				fmt.Sprintf(
					"kubectl exec pod/%s -n %s -c instana-agent -- cat /var/lib/instana/instana-agent-id",
					podName,
					cfg.Namespace(),
				),
			)

			if p.Err() == nil {
				agentID = strings.TrimSpace(p.Result())
				if agentID != "" {
					break
				}
			}

			if i < 4 {
				t.Log("Agent ID file not readable yet, waiting...")
				time.Sleep(2 * time.Second)
			} else {
				t.Logf("Failed to read agent ID from pod: %v", p.Err())
				t.Fatalf("Command output: %s", p.Result())
			}
		}

		if agentID == "" {
			t.Fatal("Agent ID is empty after all retries")
		}

		t.Logf("✓ Initial agent ID: %s on node: %s", agentID, nodeName)

		// Store the agent ID and node name in context for later comparison
		ctx = context.WithValue(ctx, "initialAgentID", agentID)
		return context.WithValue(ctx, "initialNodeName", nodeName)
	}
}

// deleteAgentPod deletes one agent pod to trigger a restart
func deleteAgentPod() features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Log("Deleting agent pod to trigger restart")

		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal("Failed to create client:", err)
		}

		// Get agent pods
		pods := &corev1.PodList{}
		listOps := resources.WithLabelSelector("app.kubernetes.io/component=instana-agent")
		err = r.List(ctx, pods, listOps)
		if err != nil || len(pods.Items) == 0 {
			t.Fatal("Error while getting agent pods:", err)
		}

		pod := pods.Items[0]
		podName := pod.Name
		t.Logf("Deleting pod: %s", podName)

		// Delete the pod
		err = r.Delete(ctx, &pod)
		if err != nil {
			t.Fatalf("Failed to delete pod: %v", err)
		}

		// Wait for the pod to be deleted
		err = wait.For(
			conditions.New(r).ResourceDeleted(&pod),
			wait.WithTimeout(2*time.Minute),
		)
		if err != nil {
			t.Fatalf("Pod was not deleted in time: %v", err)
		}

		t.Logf("✓ Pod %s deleted successfully", podName)
		return ctx
	}
}

// verifyAgentIDPersisted checks that the agent ID is the same after pod restart
func verifyAgentIDPersisted() features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Log("Verifying agent ID persisted after pod restart")

		initialAgentID, ok := ctx.Value("initialAgentID").(string)
		if !ok || initialAgentID == "" {
			t.Fatal("Initial agent ID not found in context")
		}

		initialNodeName, ok := ctx.Value("initialNodeName").(string)
		if !ok || initialNodeName == "" {
			t.Fatal("Initial node name not found in context")
		}

		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal("Failed to create client:", err)
		}

		// Get agent pods (after restart)
		pods := &corev1.PodList{}
		listOps := resources.WithLabelSelector("app.kubernetes.io/component=instana-agent")
		err = r.List(ctx, pods, listOps)
		if err != nil || len(pods.Items) == 0 {
			t.Fatal("Error while getting agent pods after restart:", err)
		}

		// Find the pod on the same node as the initial pod
		var pod *corev1.Pod
		for i := range pods.Items {
			if pods.Items[i].Spec.NodeName == initialNodeName {
				pod = &pods.Items[i]
				break
			}
		}

		if pod == nil {
			t.Fatalf("No agent pod found on the original node: %s", initialNodeName)
		}

		podName := pod.Name
		nodeName := pod.Spec.NodeName
		t.Logf(
			"Waiting for agent to persist ID in restarted pod: %s on node: %s",
			podName,
			nodeName,
		)

		// First, wait for the agent to log that it has persisted the ID
		// 6 minute timeout
		maxRetries := 72 // 72 * 5 seconds = 6 minutes
		retryInterval := 5 * time.Second
		logFound := false

		for i := 0; i < maxRetries; i++ {
			// Use kubectl logs to check for the agent ID file path log
			p := utils.RunCommand(
				fmt.Sprintf(
					"bash -c \"kubectl logs pod/%s -n %s -c instana-agent | grep 'Using agent ID file path:' || echo 'not found'\"",
					podName,
					cfg.Namespace(),
				),
			)

			output := strings.TrimSpace(p.Result())
			if p.Err() == nil && output != "not found" && output != "" {
				t.Logf("✓ Agent is using ID file path after restart: %s", output)
				logFound = true
				break
			}

			if i == 0 {
				t.Log("Waiting for agent to persist ID after restart (checking logs)...")
			}

			if i < maxRetries-1 {
				time.Sleep(retryInterval)
			}
		}

		if !logFound {
			t.Fatalf(
				"Agent did not log 'Using agent ID file path:' after restart after %d minutes",
				maxRetries*int(retryInterval.Seconds())/60,
			)
		}

		// Now read the agent ID file using kubectl exec
		t.Logf("Reading agent ID file from restarted pod: %s", podName)
		var newAgentID string

		for i := 0; i < 5; i++ { // Shorter retry since we already waited for the log
			p := utils.RunCommand(
				fmt.Sprintf(
					"kubectl exec pod/%s -n %s -c instana-agent -- cat /var/lib/instana/instana-agent-id",
					podName,
					cfg.Namespace(),
				),
			)

			if p.Err() == nil {
				newAgentID = strings.TrimSpace(p.Result())
				if newAgentID != "" {
					break
				}
			}

			if i < 4 {
				t.Log("Agent ID file not readable yet after restart, waiting...")
				time.Sleep(2 * time.Second)
			} else {
				t.Logf("Failed to read agent ID from new pod: %v", p.Err())
				t.Fatalf("Command output: %s", p.Result())
			}
		}

		if newAgentID == "" {
			t.Fatal("New agent ID is empty after all retries")
		}

		t.Logf("Initial agent ID: %s (node: %s)", initialAgentID, initialNodeName)
		t.Logf("New agent ID:     %s (node: %s)", newAgentID, nodeName)

		if initialAgentID != newAgentID {
			t.Fatalf("Agent ID changed after restart on same node! Initial: %s, New: %s",
				initialAgentID, newAgentID)
		}

		t.Logf("✓ Agent ID persisted successfully across pod restart on node: %s", nodeName)
		return ctx
	}
}

// isCloudProviderMetadataAvailable detects if cloud provider metadata is available
// When cloud provider metadata is available, the agent uses cloud provider ID
// instead of generating a MAC-based ID, which means file persistence is not used
func isCloudProviderMetadataAvailable(t *testing.T) bool {
	t.Helper()

	// Check if GCP metadata is available
	p := utils.RunCommand(
		"curl -s -m 2 http://metadata.google.internal/computeMetadata/v1/ " +
			"-H 'Metadata-Flavor: Google' || echo 'not available'",
	)
	if p.Err() == nil {
		output := strings.TrimSpace(p.Result())
		if output != "not available" && !strings.Contains(output, "Could not resolve host") {
			t.Log(
				"Detected GCP cloud provider metadata - agent will use cloud provider ID instead of MAC-based ID",
			)
			return true
		}
	}

	// Check if AWS metadata is available
	p = utils.RunCommand(
		"curl -s -m 2 http://169.254.169.254/latest/meta-data/ || echo 'not available'",
	)
	if p.Err() == nil {
		output := strings.TrimSpace(p.Result())
		if output != "not available" && !strings.Contains(output, "Could not resolve host") {
			t.Log(
				"Detected AWS cloud provider metadata - agent will use cloud provider ID instead of MAC-based ID",
			)
			return true
		}
	}

	// Check if Azure metadata is available
	p = utils.RunCommand(
		"curl -s -m 2 -H 'Metadata:true' " +
			"http://169.254.169.254/metadata/instance?api-version=2021-02-01 || echo 'not available'",
	)
	if p.Err() == nil {
		output := strings.TrimSpace(p.Result())
		if output != "not available" && !strings.Contains(output, "Could not resolve host") {
			t.Log(
				"Detected Azure cloud provider metadata - agent will use cloud provider ID instead of MAC-based ID",
			)
			return true
		}
	}

	return false
}

// Made with Bob
