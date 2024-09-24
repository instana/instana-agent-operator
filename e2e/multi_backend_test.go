/*
 * (c) Copyright IBM Corp. 2024
 * (c) Copyright Instana Inc. 2024
 */

package e2e

import (
	"context"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	log "k8s.io/klog/v2"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestMultiBackendSupport(t *testing.T) {
	installCrWithExternalSecretFeature := features.New("multiple backend support with external keyssecret").
		Setup(SetupOperatorDevBuild()).
		Setup(WaitForDeploymentToBecomeReady(InstanaOperatorDeploymentName)).
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			t.Logf("Creating dummy secret")

			err = decoder.ApplyWithManifestDir(ctx, r, "../config/samples", "external_secret_instana_agent_key.yaml", []resources.CreateOption{})
			if err != nil {
				t.Fatal(err)
			}

			t.Logf("Secret created")

			t.Logf("Creating dummy agent CR")
			err = decoder.ApplyWithManifestDir(ctx, r, "../config/samples", "instana_v1_instanaagent_multiple_backends_external_keyssecret.yaml", []resources.CreateOption{})
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("CR created")

			return ctx
		}).
		Assess("wait for k8sensor deployment to become ready", WaitForDeploymentToBecomeReady(K8sensorDeploymentName)).
		Assess("wait for agent daemonset to become ready", WaitForAgentDaemonSetToBecomeReady()).
		Assess("validate instana-agent-config secret contains 2 backends", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			log.Infof("Fetching  secret %s", InstanaAgentConfigSecretName)
			// Create a client to interact with the Kube API
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			// Check if namespace exist, otherwise just skip over it
			instanaAgentConfigSecret := &corev1.Secret{}
			err = r.Get(ctx, InstanaAgentConfigSecretName, InstanaNamespace, instanaAgentConfigSecret)
			if err != nil {
				t.Fatal("Secret could not be fetched", InstanaAgentConfigSecretName, err)
			}
			log.Info("Printing first backend config")
			firstBackendConfigString := strings.TrimSpace(string(instanaAgentConfigSecret.Data["com.instana.agent.main.sender.Backend-1.cfg"]))
			expectedFirstBackendConfigString := "host=first-backend.instana.io\nport=443\nprotocol=HTTP/2\nkey=xxx"
			log.Info(firstBackendConfigString)
			log.Info("Printing second backend config")
			secondBackendConfigString := strings.TrimSpace(string(instanaAgentConfigSecret.Data["com.instana.agent.main.sender.Backend-2.cfg"]))
			expectedSecondBackendConfigString := "host=second-backend.instana.io\nport=443\nprotocol=HTTP/2\nkey=xxx"
			log.Info(secondBackendConfigString)

			if firstBackendConfigString != expectedFirstBackendConfigString {
				t.Error("First backend does not match the expected string", firstBackendConfigString, expectedFirstBackendConfigString)
			}
			if secondBackendConfigString != expectedSecondBackendConfigString {
				t.Error("First backend does not match the expected string", secondBackendConfigString, expectedSecondBackendConfigString)
			}

			// TODO: check if file is mounted inside the pod into the right directory
			return ctx
		}).
		Feature()

	// test feature
	testEnv.Test(t, installCrWithExternalSecretFeature)
}
