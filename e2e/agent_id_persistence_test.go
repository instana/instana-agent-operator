/*
 * (c) Copyright IBM Corp. 2026
 */

package e2e

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

// TestAgentIDPersistence verifies that the agent ID persists across pod restarts
// when INSTANA_PERSIST_HOST_UNIQUE_ID is set and /var/lib is mounted
func TestAgentIDPersistence(t *testing.T) {
	agent := NewAgentCr()

	agentIDPersistenceFeature := features.New("agent ID persistence across pod restarts").
		Setup(SetupOperatorDevBuild()).
		Setup(DeployAgentCr(&agent)).
		Assess("wait for instana-agent-controller-manager deployment to become ready",
			WaitForDeploymentToBecomeReady(InstanaOperatorDeploymentName)).
		Assess("wait for k8sensor deployment to become ready",
			WaitForDeploymentToBecomeReady(K8sensorDeploymentName)).
		Assess("wait for agent daemonset to become ready",
			WaitForAgentDaemonSetToBecomeReady()).
		Assess("verify INSTANA_PERSIST_HOST_UNIQUE_ID env var is set",
			verifyPersistHostUniqueIDEnvVar()).
		Assess("get initial agent ID from pod",
			getAndStoreAgentID()).
		Assess("delete agent pod to trigger restart",
			deleteAgentPod()).
		Assess("wait for agent daemonset to become ready after restart",
			WaitForAgentDaemonSetToBecomeReady()).
		Assess("verify agent ID persisted after restart",
			verifyAgentIDPersisted()).
		Feature()

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
		t.Logf("Reading agent ID from pod: %s", pod.Name)

		// Wait for the agent to create the ID file (retry for up to 2 minutes)
		var agentID string
		maxRetries := 24 // 24 * 5 seconds = 2 minutes
		for i := 0; i < maxRetries; i++ {
			var stdout, stderr bytes.Buffer
			err := r.ExecInPod(
				ctx,
				cfg.Namespace(),
				pod.Name,
				"instana-agent",
				[]string{"cat", "/var/lib/instana/instana-agent-id"},
				&stdout,
				&stderr,
			)

			if err == nil {
				agentID = strings.TrimSpace(stdout.String())
				if agentID != "" {
					break
				}
			}

			if i == 0 {
				t.Log("Agent ID file not yet created, waiting...")
			}

			if i == maxRetries-1 {
				t.Log(stderr.String())
				t.Fatalf("Failed to read agent ID from pod after %d retries: %v", maxRetries, err)
			}

			time.Sleep(5 * time.Second)
		}

		if agentID == "" {
			t.Fatal("Agent ID is empty after all retries")
		}

		t.Logf("✓ Initial agent ID: %s", agentID)

		// Store the agent ID in context for later comparison
		return context.WithValue(ctx, "initialAgentID", agentID)
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

		pod := pods.Items[0]
		t.Logf("Reading agent ID from new pod: %s", pod.Name)

		// Wait for the agent to create/read the ID file (retry for up to 2 minutes)
		var newAgentID string
		maxRetries := 24 // 24 * 5 seconds = 2 minutes
		for i := 0; i < maxRetries; i++ {
			var stdout, stderr bytes.Buffer
			err := r.ExecInPod(
				ctx,
				cfg.Namespace(),
				pod.Name,
				"instana-agent",
				[]string{"cat", "/var/lib/instana/instana-agent-id"},
				&stdout,
				&stderr,
			)

			if err == nil {
				newAgentID = strings.TrimSpace(stdout.String())
				if newAgentID != "" {
					break
				}
			}

			if i == 0 {
				t.Log("Agent ID file not yet available in new pod, waiting...")
			}

			if i == maxRetries-1 {
				t.Log(stderr.String())
				t.Fatalf(
					"Failed to read agent ID from new pod after %d retries: %v",
					maxRetries,
					err,
				)
			}

			time.Sleep(5 * time.Second)
		}

		if newAgentID == "" {
			t.Fatal("New agent ID is empty after all retries")
		}

		t.Logf("Initial agent ID: %s", initialAgentID)
		t.Logf("New agent ID:     %s", newAgentID)

		if initialAgentID != newAgentID {
			t.Fatalf("Agent ID changed after restart! Initial: %s, New: %s",
				initialAgentID, newAgentID)
		}

		t.Log("✓ Agent ID persisted successfully across pod restart")
		return ctx
	}
}

// Made with Bob
