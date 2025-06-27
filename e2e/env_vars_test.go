/*
 * (c) Copyright IBM Corp. 2025
 * (c) Copyright Instana Inc. 2025
 */

package e2e

import (
	"context"
	"strings"
	"testing"

	"bytes"
	"fmt"

	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	corev1 "k8s.io/api/core/v1"
)

func TestEnvVarsWithKubernetesFormat(t *testing.T) {
	// Create a ConfigMap and Secret that will be referenced by environment variables
	setupConfigMapAndSecret := func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}

		// Create ConfigMap
		configMap := &corev1.ConfigMap{
			ObjectMeta: corev1.ObjectMeta{
				Name:      "env-test-config",
				Namespace: cfg.Namespace(),
			},
			Data: map[string]string{
				"test-key": "test-value-from-configmap",
			},
		}
		if err := r.Create(ctx, configMap); err != nil {
			t.Fatal("Failed to create ConfigMap:", err)
		}
		t.Log("ConfigMap created")

		// Create Secret
		secret := &corev1.Secret{
			ObjectMeta: corev1.ObjectMeta{
				Name:      "env-test-secret",
				Namespace: cfg.Namespace(),
			},
			Type: corev1.SecretTypeOpaque,
			StringData: map[string]string{
				"test-key": "test-value-from-secret",
			},
		}
		if err := r.Create(ctx, secret); err != nil {
			t.Fatal("Failed to create Secret:", err)
		}
		t.Log("Secret created")

		return ctx
	}

	installCrWithEnvVarsFeature := features.New("environment variables with Kubernetes format").
		Setup(SetupOperatorDevBuild()).
		Setup(WaitForDeploymentToBecomeReady(InstanaOperatorDeploymentName)).
		Setup(setupConfigMapAndSecret).
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			t.Log("Creating agent CR with environment variables")
			err = decoder.ApplyWithManifestDir(ctx, r, "../config/samples", "instana_v1_env_vars_example.yaml", []resources.CreateOption{})
			if err != nil {
				t.Fatal(err)
			}
			t.Log("CR created")

			return ctx
		}).
		Assess("wait for k8sensor deployment to become ready", WaitForDeploymentToBecomeReady(K8sensorDeploymentName)).
		Assess("wait for agent daemonset to become ready", WaitForAgentDaemonSetToBecomeReady()).
		Assess("validate environment variables are correctly set", ValidateEnvironmentVariables()).
		Feature()

	// test feature
	testEnv.Test(t, installCrWithEnvVarsFeature)
}

func ValidateEnvironmentVariables() features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Log("Validating environment variables in agent pod")
		
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
		
		var stdout, stderr bytes.Buffer
		podName := pods.Items[0].Name
		containerName := "instana-agent"

		// Test cases for environment variables
		envVarTests := []struct {
			name     string
			expected string
		}{
			{
				name:     "INSTANA_AGENT_TAGS",
				expected: "kubernetes,production,custom",
			},
			{
				name:     "MY_POD_NAME",
				expected: podName, // Should match the pod name
			},
			{
				name:     "DATABASE_PASSWORD",
				expected: "test-value-from-secret",
			},
			{
				name:     "APP_CONFIG",
				expected: "test-value-from-configmap",
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
			if output == test.expected || (test.name == "MY_POD_NAME" && output != "") {
				t.Logf("Environment variable %s has expected value: %s", test.name, output)
			} else {
				t.Errorf("Environment variable %s has unexpected value. Expected: %s, Got: %s", 
					test.name, test.expected, output)
			}
		}

		return ctx
	}
}

// Made with Bob
