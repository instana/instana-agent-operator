package e2e

import (
	"context"
	"strings"
	"testing"

	"bytes"
	"fmt"

	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
)

func TestSecureETCDScraping(t *testing.T) {
	// Create a CA certificate secret that will be used for ETCD scraping
	setupETCDCASecret := func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}

		// Create CA certificate Secret
		caSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "etcd-ca-cert",
				Namespace: cfg.Namespace(),
			},
			Type: corev1.SecretTypeOpaque,
			StringData: map[string]string{
				"ca.crt": "-----BEGIN CERTIFICATE-----\n" +
					"MIIBVzCB/qADAgECAhBsL8JXhPlGrvu/hMCRWXAFMAoGCCqGSM49BAMCMBkxFzAV\n" +
					"BgNVBAMTDmV0Y2QtY2EtdGVzdGluZzAeFw0yMzA1MDEwMDAwMDBaFw0zMzA1MDEw\n" +
					"MDAwMDBaMBkxFzAVBgNVBAMTDmV0Y2QtY2EtdGVzdGluZzBZMBMGByqGSM49AgEG\n" +
					"CCqGSM49AwEHA0IABLnVZEsHdWyq0QgOsS7E5RgXIrBVnL+bZtDEkqkW/kD3N4Fm\n" +
					"JUK1MdJzwQ7QKnNUQbwpLHmp7vZPMlWxwhPzrMyjQjBAMA4GA1UdDwEB/wQEAwIC\n" +
					"pDAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBQxCIIkW0uFSp+1+EwcOWeyZkKJ\n" +
					"RTAKBggqhkjOPQQDAgNHADBEAiAWYHwQPZPXULYcGXNpEPE0feKOO9iq9TwH44j5\n" +
					"EAQRJgIgLnJBGd4ZS8+H6TS6WbkQ9MKX1jgCt0QlPGOo+ZxpnBs=\n" +
					"-----END CERTIFICATE-----",
			},
		}
		if err := r.Create(ctx, caSecret); err != nil {
			t.Fatal("Failed to create CA certificate Secret:", err)
		}
		t.Log("CA certificate Secret created")

		return ctx
	}

	installCrWithSecureETCDFeature := features.New("secure ETCD scraping").
		Setup(SetupOperatorDevBuild()).
		Setup(WaitForDeploymentToBecomeReady(InstanaOperatorDeploymentName)).
		Setup(setupETCDCASecret).
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			t.Log("Creating agent CR with secure ETCD configuration")
			err = decoder.ApplyWithManifestDir(
				ctx,
				r,
				"../config/samples",
				"instana_v1_agent.yaml",
				[]resources.CreateOption{},
			)
			if err != nil {
				t.Fatal(err)
			}

			// Patch the agent CR to add ETCD configuration
			agent := &instanav1.InstanaAgent{}
			err = r.Get(ctx, "instana-agent", cfg.Namespace(), agent)
			if err != nil {
				t.Fatal("Failed to get agent CR:", err)
			}

			// Apply patch to add ETCD configuration
			patchData := []byte(`{
				"spec": {
					"k8s_sensor": {
						"etcd": {
							"insecure": false,
							"ca": {
								"mountPath": "/etc/ssl/certs",
								"secretName": "etcd-ca-cert"
							}
						},
						"restClient": {
							"hostAllowlist": ["kubernetes.default.svc"],
							"ca": {
								"mountPath": "/etc/ssl/control-plane",
								"secretName": "etcd-ca-cert"
							}
						}
					}
				}
			}`)
			err = r.Patch(ctx, agent, k8s.Patch{
				PatchType: types.MergePatchType,
				Data:      patchData,
			})
			if err != nil {
				t.Fatal("Failed to patch agent CR:", err)
			}

			t.Log("CR created and patched with secure ETCD configuration")
			return ctx
		}).
		Assess("wait for k8sensor deployment to become ready", WaitForDeploymentToBecomeReady(K8sensorDeploymentName)).
		Assess("wait for agent daemonset to become ready", WaitForAgentDaemonSetToBecomeReady()).
		Assess("validate ETCD CA certificate is mounted", ValidateETCDCAMounted()).
		Assess("validate ETCD environment variables are set", ValidateETCDEnvironmentVariables()).
		Feature()

	// test feature
	testEnv.Test(t, installCrWithSecureETCDFeature)
}

func ValidateETCDCAMounted() features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Log("Validating ETCD CA certificate is mounted in k8sensor pod")

		// Create a client to interact with the Kube API
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}

		// Get k8sensor pods
		pods := &corev1.PodList{}
		listOps := resources.WithLabelSelector("app=k8sensor")
		err = r.List(ctx, pods, listOps)
		if err != nil || len(pods.Items) == 0 {
			t.Fatal("Error while getting k8sensor pods:", err)
		}

		var stdout, stderr bytes.Buffer
		podName := pods.Items[0].Name
		containerName := "instana-agent"

		// Check if the CA certificate file exists
		if err := r.ExecInPod(
			ctx,
			cfg.Namespace(),
			podName,
			containerName,
			[]string{"ls", "-la", "/etc/ssl/certs/ca.crt"},
			&stdout,
			&stderr,
		); err != nil {
			t.Log(stderr.String())
			t.Fatal("Failed to execute command in pod:", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "/etc/ssl/certs/ca.crt") {
			t.Errorf("CA certificate file not found: %s", output)
		} else {
			t.Logf("CA certificate file found: %s", output)
		}

		return ctx
	}
}

func ValidateETCDEnvironmentVariables() features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Log("Validating ETCD environment variables in k8sensor pod")

		// Create a client to interact with the Kube API
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}

		// Get k8sensor pods
		pods := &corev1.PodList{}
		listOps := resources.WithLabelSelector("app=k8sensor")
		err = r.List(ctx, pods, listOps)
		if err != nil || len(pods.Items) == 0 {
			t.Fatal("Error while getting k8sensor pods:", err)
		}

		var stdout, stderr bytes.Buffer
		podName := pods.Items[0].Name
		containerName := "instana-agent"

		// Test cases for environment variables
		envVarTests := []struct {
			name     string
			expected string
		}{
			{
				name:     "ETCD_CA_FILE",
				expected: "/etc/ssl/certs/ca.crt",
			},
			{
				name:     "ETCD_INSECURE",
				expected: "false",
			},
			{
				name:     "CONTROL_PLANE_CA_FILE",
				expected: "/etc/ssl/control-plane/ca.crt",
			},
			{
				name:     "REST_CLIENT_HOST_ALLOWLIST",
				expected: "kubernetes.default.svc",
			},
		}

		for _, test := range envVarTests {
			stdout.Reset()
			stderr.Reset()

			// Execute command to print environment variable
			if err := r.ExecInPod(
				ctx,
				cfg.Namespace(),
				podName,
				containerName,
				[]string{"sh", "-c", fmt.Sprintf("echo $%s", test.name)},
				&stdout,
				&stderr,
			); err != nil {
				t.Log(stderr.String())
				t.Fatal("Failed to execute command in pod:", err)
			}

			output := strings.TrimSpace(stdout.String())
			if output == test.expected {
				t.Logf("Environment variable %s has expected value: %s", test.name, output)
			} else {
				t.Errorf("Environment variable %s has unexpected value. Expected: %s, Got: %s",
					test.name, test.expected, output)
			}
		}

		return ctx
	}
}
