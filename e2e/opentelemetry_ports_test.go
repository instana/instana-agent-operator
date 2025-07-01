/*
 * (c) Copyright IBM Corp. 2025
 */

package e2e

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/pointer"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	portCheckTimeout  = 2 * time.Minute
	portCheckInterval = 5 * time.Second
)

func TestOpenTelemetryPorts(t *testing.T) {
	// Define the test feature
	openTelemetryPortsFeature := features.New("opentelemetry ports configuration").
		Setup(SetupOperatorDevBuild()).
		Setup(WaitForDeploymentToBecomeReady(InstanaOperatorDeploymentName)).
		Setup(CreateAgentWithDefaultPorts()).
		Assess("wait for k8sensor deployment to become ready", WaitForDeploymentToBecomeReady(K8sensorDeploymentName)).
		Assess("wait for agent daemonset to become ready", WaitForAgentDaemonSetToBecomeReady()).
		Assess("validate default OpenTelemetry ports are used", ValidateDefaultOpenTelemetryPorts()).
		Assess("update agent CR with custom ports", UpdateAgentWithCustomPorts()).
		Assess("wait for agent daemonset to become ready after update", WaitForAgentDaemonSetToBecomeReady()).
		Assess("validate custom OpenTelemetry ports are used", ValidateCustomOpenTelemetryPorts()).
		Feature()

	// Run the test
	testEnv.Test(t, openTelemetryPortsFeature)
}

// CreateAgentWithDefaultPorts creates an agent CR with default OpenTelemetry port configuration
func CreateAgentWithDefaultPorts() features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}

		// Create a basic agent CR with default OpenTelemetry configuration
		agent := NewAgentCr()
		// Ensure OpenTelemetry is enabled with default settings
		agent.Spec.OpenTelemetry.Enabled.Enabled = pointer.To(true)

		if err := r.Create(ctx, &agent); err != nil {
			t.Fatal("Failed to create agent CR:", err)
		}
		t.Log("Agent CR created with default OpenTelemetry ports")

		return ctx
	}
}

// ValidateDefaultOpenTelemetryPorts checks if the default OpenTelemetry ports (4317 and 4318) are being used
func ValidateDefaultOpenTelemetryPorts() features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Log("Validating default OpenTelemetry ports in agent pod")

		// Create a client to interact with the Kube API
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}

		// Get agent pods
		pods := &corev1.PodList{}
		listOps := resources.WithLabelSelector("app.kubernetes.io/component=instana-agent")
		err = r.List(ctx, pods, listOps)
		if err != nil || len(pods.Items) == 0 {
			t.Fatal("Error while getting agent pods:", err)
		}

		podName := pods.Items[0].Name
		containerName := "instana-agent"

		// Check for GRPC port 4317 with retry
		t.Log("Checking for GRPC port 4317 with retry")
		checkPortWithRetry(ctx, t, r, cfg.Namespace(), podName, containerName, "4317", "GRPC")

		// Check for HTTP port 4318 with retry
		t.Log("Checking for HTTP port 4318 with retry")
		checkPortWithRetry(ctx, t, r, cfg.Namespace(), podName, containerName, "4318", "HTTP")

		return ctx
	}
}

// Helper function to check for a port with retry logic
func checkPortWithRetry(ctx context.Context, t *testing.T, r *resources.Resources, namespace, podName, containerName, port, portType string) {
	startTime := time.Now()
	deadline := startTime.Add(portCheckTimeout)

	for time.Now().Before(deadline) {
		var stdout, stderr bytes.Buffer

		err := r.ExecInPod(
			ctx,
			namespace,
			podName,
			containerName,
			[]string{"sh", "-c", "ss -tulnp | grep " + port},
			&stdout,
			&stderr,
		)

		output := stdout.String()
		if err == nil && strings.Contains(output, port) {
			t.Logf("%s port %s found after %v. Output: %s",
				portType, port, time.Since(startTime), output)
			return
		}

		t.Logf("%s port %s not found yet, retrying in %v. Error: %v",
			portType, port, portCheckInterval, err)
		time.Sleep(portCheckInterval)
	}

	t.Fatalf("%s port %s not found after %v timeout", portType, port, portCheckTimeout)
}

// UpdateAgentWithCustomPorts updates the agent CR to use custom OpenTelemetry ports
func UpdateAgentWithCustomPorts() features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}

		// Get the current agent CR
		agent := &instanav1.InstanaAgent{}
		if err := r.Get(ctx, "instana-agent", cfg.Namespace(), agent); err != nil {
			t.Fatal("Failed to get agent CR:", err)
		}

		// Update the agent CR with custom ports (default + 10)
		agent.Spec.OpenTelemetry.GRPC.Port = pointer.To(int32(4327))
		agent.Spec.OpenTelemetry.HTTP.Port = pointer.To(int32(4328))

		if err := r.Update(ctx, agent); err != nil {
			t.Fatal("Failed to update agent CR:", err)
		}
		t.Log("Agent CR updated with custom OpenTelemetry ports")

		return ctx
	}
}

// ValidateCustomOpenTelemetryPorts checks if the custom OpenTelemetry ports are being used and the default ones are not
func ValidateCustomOpenTelemetryPorts() features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Log("Validating custom OpenTelemetry ports in agent pod")

		// Create a client to interact with the Kube API
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}

		// Get agent pods
		pods := &corev1.PodList{}
		listOps := resources.WithLabelSelector("app.kubernetes.io/component=instana-agent")
		err = r.List(ctx, pods, listOps)
		if err != nil || len(pods.Items) == 0 {
			t.Fatal("Error while getting agent pods:", err)
		}

		podName := pods.Items[0].Name
		containerName := "instana-agent"

		// Verify old ports are no longer used
		verifyPortNotInUse(ctx, t, r, cfg.Namespace(), podName, containerName, "4317", "old GRPC")
		verifyPortNotInUse(ctx, t, r, cfg.Namespace(), podName, containerName, "4318", "old HTTP")

		// Check for new ports with retry
		t.Log("Checking for new GRPC port 4327 with retry")
		checkPortWithRetry(ctx, t, r, cfg.Namespace(), podName, containerName, "4327", "new GRPC")

		t.Log("Checking for new HTTP port 4328 with retry")
		checkPortWithRetry(ctx, t, r, cfg.Namespace(), podName, containerName, "4328", "new HTTP")

		return ctx
	}
}

// Helper function to verify a port is not in use
func verifyPortNotInUse(ctx context.Context, t *testing.T, r *resources.Resources, namespace, podName, containerName, port, portType string) {
	startTime := time.Now()
	deadline := startTime.Add(portCheckTimeout)

	// First wait a bit to ensure agent has had time to switch ports
	time.Sleep(10 * time.Second)

	for time.Now().Before(deadline) {
		var stdout, stderr bytes.Buffer

		err := r.ExecInPod(
			ctx,
			namespace,
			podName,
			containerName,
			[]string{"sh", "-c", "ss -tulnp | grep " + port + " || echo 'Port not found'"},
			&stdout,
			&stderr,
		)

		if err != nil {
			t.Logf("Error executing command: %v", err)
			time.Sleep(portCheckInterval)
			continue
		}

		output := stdout.String()
		if strings.Contains(output, "Port not found") {
			t.Logf("%s port %s confirmed not in use after %v",
				portType, port, time.Since(startTime))
			return
		}

		t.Logf("%s port %s still in use, waiting for it to be released. Retrying in %v",
			portType, port, portCheckInterval)
		time.Sleep(portCheckInterval)
	}

	t.Fatalf("%s port %s still in use after %v timeout", portType, port, portCheckTimeout)
}
