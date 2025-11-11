/*
 * (c) Copyright IBM Corp. 2024
 * (c) Copyright Instana Inc. 2024
 */

package e2e

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestExtraVolumeWithSecret(t *testing.T) {
	installCrWithExtraVolumeFeature := features.New("extra volume with secret").
		Setup(SetupOperatorDevBuild()).
		Setup(WaitForDeploymentToBecomeReady(InstanaOperatorDeploymentName)).
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			t.Logf("Creating dummy secrets and configmap")

			err = decoder.ApplyWithManifestDir(ctx, r, "../config/samples", "external_secret_instana_agent_key.yaml", []resources.CreateOption{})
			if err != nil {
				t.Fatal(err)
			}
			CleanupSecretAfterTest(t, cfg.Namespace(), "instana-agent-key")

			// Create the my-secret Secret referenced in the environment variables
			mySecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-secret",
					Namespace: cfg.Namespace(),
				},
				Type: corev1.SecretTypeOpaque,
				StringData: map[string]string{
					"password": "test-password",
				},
			}
			if err := r.Create(ctx, mySecret); err != nil {
				t.Fatal("Failed to create my-secret:", err)
			}
			CleanupSecretAfterTest(t, cfg.Namespace(), "my-secret")

			// Create the my-config ConfigMap referenced in the environment variables
			myConfig := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-config",
					Namespace: cfg.Namespace(),
				},
				Data: map[string]string{
					"value": "test-value",
				},
			}
			if err := r.Create(ctx, myConfig); err != nil {
				t.Fatal("Failed to create my-config:", err)
			}
			CleanupConfigMapAfterTest(t, cfg.Namespace(), "my-config")

			t.Logf("Secrets and ConfigMap created")

			t.Logf("Creating dummy agent CR with extra volume")
			err = decoder.ApplyWithManifestDir(ctx, r, "../config/samples", "instana_v1_extended_instanaagent.yaml", []resources.CreateOption{})
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("CR created")

			return ctx
		}).
		Assess("wait for first k8sensor deployment to become ready", WaitForDeploymentToBecomeReady(K8sensorDeploymentName)).
		Assess("wait for agent daemonset to become ready", WaitForAgentDaemonSetToBecomeReady()).
		Assess("validate secret files are created from extra mounted volume", ValidateSecretsMountedFromExtraVolume()).
		Feature()

	// test feature
	testEnv.Test(t, installCrWithExtraVolumeFeature)
}
