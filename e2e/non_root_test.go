/*
 * (c) Copyright IBM Corp. 2026
 */

package e2e

import (
	"bytes"
	"context"
	"strings"
	"testing"

	v1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/pointer"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

// nonRootCapabilities is the expected set of capabilities when nonRoot.enabled=true,
// mirroring the operator's nonRootCapabilities var in daemonset.go.
var expectedNonRootCapabilities = []string{
	"AUDIT_WRITE",
	"DAC_READ_SEARCH",
	"DAC_OVERRIDE",
	"FOWNER",
	"KILL",
	"NET_ADMIN",
	"NET_RAW",
	"SETPCAP",
	"SETUID",
	"SETGID",
	"SYS_ADMIN",
	"SYS_CHROOT",
	"SYS_PTRACE",
	"SYS_RESOURCE",
	"SYSLOG",
}

func TestNonRootMode(t *testing.T) {
	CollectOperatorLogsOnFailure(t)

	agent := NewAgentCr()
	agent.Spec.Agent.NonRoot = v1.NonRootSpec{
		Enabled: pointer.To(true),
	}

	nonRootFeature := features.New("non-root agent mode").
		Setup(SetupOperatorDevBuild()).
		Setup(DeployAgentCr(&agent)).
		Assess("wait for k8sensor deployment to become ready", WaitForDeploymentToBecomeReady(K8sensorDeploymentName)).
		Assess("wait for agent daemonset to become ready", WaitForAgentDaemonSetToBecomeReady()).
		Assess("verify daemonset security context has capabilities not privileged", ValidateNonRootSecurityContext()).
		Assess("verify INSTANA_AGENT_NON_ROOT env var is set to true", ValidateNonRootEnvVar()).
		Assess("check agent log for successful connection", WaitForAgentSuccessfulBackendConnection()).
		Feature()

	testEnv.Test(t, nonRootFeature)
}

// ValidateNonRootSecurityContext verifies that the agent container SecurityContext has:
//   - privileged: false
//   - allowPrivilegeEscalation: true
//   - capabilities.add containing all 15 required non-root capabilities
func ValidateNonRootSecurityContext() features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Log("Validating non-root SecurityContext on agent pod")

		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}

		pods := &corev1.PodList{}
		if err := r.List(
			ctx,
			pods,
			resources.WithLabelSelector("app.kubernetes.io/component=instana-agent"),
		); err != nil ||
			len(pods.Items) == 0 {
			t.Fatal("Error getting agent pods:", err)
		}

		sc := pods.Items[0].Spec.Containers[0].SecurityContext
		if sc == nil {
			t.Fatal("SecurityContext is nil on agent container")
		}

		// privileged must be false
		if sc.Privileged == nil || *sc.Privileged {
			t.Errorf("Expected privileged=false, got %v", sc.Privileged)
		} else {
			t.Log("privileged=false ✓")
		}

		// allowPrivilegeEscalation must be true (required for setpriv --ambient-caps)
		if sc.AllowPrivilegeEscalation == nil || !*sc.AllowPrivilegeEscalation {
			t.Errorf("Expected allowPrivilegeEscalation=true, got %v", sc.AllowPrivilegeEscalation)
		} else {
			t.Log("allowPrivilegeEscalation=true ✓")
		}

		// all 15 expected capabilities must be present
		if sc.Capabilities == nil {
			t.Fatal("Capabilities is nil on agent container SecurityContext")
		}
		addedCaps := make(map[string]bool, len(sc.Capabilities.Add))
		for _, c := range sc.Capabilities.Add {
			addedCaps[string(c)] = true
		}
		for _, expected := range expectedNonRootCapabilities {
			if !addedCaps[expected] {
				t.Errorf(
					"Expected capability %s not found in capabilities.add: %v",
					expected,
					sc.Capabilities.Add,
				)
			}
		}
		t.Logf("All %d expected capabilities present ✓", len(expectedNonRootCapabilities))

		return ctx
	}
}

// ValidateNonRootEnvVar verifies INSTANA_AGENT_NON_ROOT=true is set in the agent container.
func ValidateNonRootEnvVar() features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Log("Validating INSTANA_AGENT_NON_ROOT env var in agent pod")

		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}

		pods := &corev1.PodList{}
		if err := r.List(
			ctx,
			pods,
			resources.WithLabelSelector("app.kubernetes.io/component=instana-agent"),
		); err != nil ||
			len(pods.Items) == 0 {
			t.Fatal("Error getting agent pods:", err)
		}

		var stdout, stderr bytes.Buffer
		if err := r.ExecInPod(
			ctx,
			cfg.Namespace(),
			pods.Items[0].Name,
			"instana-agent",
			[]string{"sh", "-c", "echo $INSTANA_AGENT_NON_ROOT"},
			&stdout,
			&stderr,
		); err != nil {
			t.Log(stderr.String())
			t.Fatal("Failed to exec in pod:", err)
		}

		val := strings.TrimSpace(stdout.String())
		if val != "true" {
			t.Errorf("Expected INSTANA_AGENT_NON_ROOT=true, got %q", val)
		} else {
			t.Logf("INSTANA_AGENT_NON_ROOT=%s ✓", val)
		}

		// also verify the agent process is running as non-root user
		stdout.Reset()
		stderr.Reset()
		if err := r.ExecInPod(
			ctx,
			cfg.Namespace(),
			pods.Items[0].Name,
			"instana-agent",
			[]string{"sh", "-c", "stat -c %U /proc/$(pgrep -o -f 'karaf.home=.*instana')"},
			&stdout,
			&stderr,
		); err == nil {
			user := strings.TrimSpace(stdout.String())
			if user == "root" {
				t.Errorf("Agent process is running as root, expected non-root user")
			} else {
				t.Logf("Agent process running as non-root user: %s ✓", user)
			}
		}

		return ctx
	}
}
