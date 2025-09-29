/*
(c) Copyright IBM Corp. 2025

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
	"sigs.k8s.io/e2e-framework/pkg/types"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
)

const (
	// expectedCAAbsolutePath is the path where the CA certificate is mounted in the k8sensor pod
	expectedCAAbsolutePath = "/etc/ssl/certs/ca.crt"

	// testCACertificateContent is the CA certificate content used in tests
	testCACertificateContent = "-----BEGIN CERTIFICATE-----\n" +
		"MIIBVzCB/qADAgECAhBsL8JXhPlGrvu/hMCRWXAFMAoGCCqGSM49BAMCMBkxFzAV\n" +
		"BgNVBAMTDmV0Y2QtY2EtdGVzdGluZzAeFw0yMzA1MDEwMDAwMDBaFw0zMzA1MDEw\n" +
		"MDAwMDBaMBkxFzAVBgNVBAMTDmV0Y2QtY2EtdGVzdGluZzBZMBMGByqGSM49AgEG\n" +
		"CCqGSM49AwEHA0IABLnVZEsHdWyq0QgOsS7E5RgXIrBVnL+bZtDEkqkW/kD3N4Fm\n" +
		"JUK1MdJzwQ7QKnNUQbwpLHmp7vZPMlWxwhPzrMyjQjBAMA4GA1UdDwEB/wQEAwIC\n" +
		"pDAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBQxCIIkW0uFSp+1+EwcOWeyZkKJ\n" +
		"RTAKBggqhkjOPQQDAgNHADBEAiAWYHwQPZPXULYcGXNpEPE0feKOO9iq9TwH44j5\n" +
		"EAQRJgIgLnJBGd4ZS8+H6TS6WbkQ9MKX1jgCt0QlPGOo+ZxpnBs=\n" +
		"-----END CERTIFICATE-----"
)

func SetupETCDCASecret() types.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}

		// Create CA certificate Secret
		caSecret := &corev1.Secret{ // pragma: allowlist secret
			ObjectMeta: metav1.ObjectMeta{
				Name:      "etcd-ca-cert",
				Namespace: cfg.Namespace(),
			},
			Type: corev1.SecretTypeOpaque,
			StringData: map[string]string{
				"ca.crt": testCACertificateContent,
			},
		}
		if err := r.Create(ctx, caSecret); err != nil {
			t.Fatal("Failed to create CA certificate Secret:", err) // pragma: allowlist secret
		}
		t.Log("CA certificate Secret created")

		return ctx
	}
}

func SetupInstanaAgentCR() types.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
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
			PatchType: k8stypes.MergePatchType,
			Data:      patchData,
		})
		if err != nil {
			t.Fatal("Failed to patch agent CR:", err)
		}

		t.Log("CR created and patched with secure ETCD configuration")
		return ctx
	}
}

func TestSecureETCDScraping(t *testing.T) {

	installCrWithSecureETCDFeature := features.New("secure ETCD scraping").
		Setup(SetupOperatorDevBuild()).
		Setup(WaitForDeploymentToBecomeReady(InstanaOperatorDeploymentName)).
		Setup(SetupETCDCASecret()).
		Setup(SetupInstanaAgentCR()).
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

		// Check if the CA certificate file exists and validate its content
		if err := r.ExecInPod(
			ctx,
			cfg.Namespace(),
			podName,
			containerName,
			[]string{"cat", expectedCAAbsolutePath},
			&stdout,
			&stderr,
		); err != nil {
			t.Log(stderr.String())
			t.Fatal("Failed to execute command in pod:", err)
		}

		output := stdout.String()
		if strings.TrimSpace(output) != strings.TrimSpace(testCACertificateContent) {
			t.Errorf(
				"CA certificate content does not match expected.\nExpected:\n%s\nActual:\n%s",
				testCACertificateContent,
				output,
			)
		} else {
			t.Logf("CA certificate content matches expected at path: %s", expectedCAAbsolutePath)
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
				expected: expectedCAAbsolutePath,
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
