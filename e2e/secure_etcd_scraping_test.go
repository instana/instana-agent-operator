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
	"path/filepath"
	"strings"
	"testing"

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
			"instana_v1_instanaagent.yaml",
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

		pod := pods.Items[0]
		containerName := "instana-agent"

		var k8sensorContainer *corev1.Container
		for i := range pod.Spec.Containers {
			if pod.Spec.Containers[i].Name == containerName {
				k8sensorContainer = &pod.Spec.Containers[i]
				break
			}
		}
		if k8sensorContainer == nil {
			t.Fatalf("Container %s not found in pod %s", containerName, pod.Name)
		}

		expectedMountPath := filepath.Dir(expectedCAAbsolutePath)
		var mount *corev1.VolumeMount
		for i := range k8sensorContainer.VolumeMounts {
			if k8sensorContainer.VolumeMounts[i].MountPath == expectedMountPath {
				mount = &k8sensorContainer.VolumeMounts[i]
				break
			}
		}
		if mount == nil {
			t.Fatalf(
				"Volume mount with path %s not found on container %s",
				expectedMountPath,
				containerName,
			)
		}

		var volume *corev1.Volume
		for i := range pod.Spec.Volumes {
			if pod.Spec.Volumes[i].Name == mount.Name {
				volume = &pod.Spec.Volumes[i]
				break
			}
		}
		if volume == nil {
			t.Fatalf(
				"Volume %s referenced by mount %s not found on pod %s",
				mount.Name,
				expectedMountPath,
				pod.Name,
			)
		}
		if volume.Secret == nil {
			t.Fatalf("Volume %s is not backed by a secret", mount.Name)
		}
		if volume.Secret.SecretName != "etcd-ca-cert" {
			t.Errorf(
				"Volume %s references secret %s, expected etcd-ca-cert",
				mount.Name,
				volume.Secret.SecretName,
			)
		} else {
			t.Logf("Volume %s correctly references secret etcd-ca-cert", mount.Name)
		}

		caSecret := &corev1.Secret{}
		if err := r.Get(ctx, "etcd-ca-cert", cfg.Namespace(), caSecret); err != nil {
			t.Fatal("Failed to fetch CA secret:", err)
		}
		secretContent, ok := caSecret.Data["ca.crt"]
		if !ok {
			t.Fatal("CA secret does not contain ca.crt key")
		}
		if strings.TrimSpace(string(secretContent)) != strings.TrimSpace(testCACertificateContent) {
			t.Errorf(
				"CA secret content does not match expected.\nExpected:\n%s\nActual:\n%s",
				testCACertificateContent,
				string(secretContent),
			)
		} else {
			t.Logf("CA secret content matches expected value")
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

		pod := pods.Items[0]
		containerName := "instana-agent"

		var k8sensorContainer *corev1.Container
		for i := range pod.Spec.Containers {
			if pod.Spec.Containers[i].Name == containerName {
				k8sensorContainer = &pod.Spec.Containers[i]
				break
			}
		}
		if k8sensorContainer == nil {
			t.Fatalf("Container %s not found in pod %s", containerName, pod.Name)
		}

		expectedEnvs := map[string]string{
			"ETCD_CA_FILE":               expectedCAAbsolutePath,
			"ETCD_INSECURE":              "false",
			"CONTROL_PLANE_CA_FILE":      "/etc/ssl/control-plane/ca.crt",
			"REST_CLIENT_HOST_ALLOWLIST": "kubernetes.default.svc",
		}

		found := make(map[string]bool, len(expectedEnvs))
		for _, envVar := range k8sensorContainer.Env {
			if expectedValue, ok := expectedEnvs[envVar.Name]; ok {
				if envVar.Value == expectedValue {
					t.Logf(
						"Environment variable %s has expected value: %s",
						envVar.Name,
						envVar.Value,
					)
				} else {
					t.Errorf(
						"Environment variable %s has unexpected value. Expected: %s, Got: %s",
						envVar.Name,
						expectedValue,
						envVar.Value,
					)
				}
				found[envVar.Name] = true
			}
		}
		for name := range expectedEnvs {
			if !found[name] {
				t.Errorf("Environment variable %s not found on container %s", name, containerName)
			}
		}

		return ctx
	}
}
