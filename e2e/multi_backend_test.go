/*
 * (c) Copyright IBM Corp. 2024
 * (c) Copyright Instana Inc. 2024
 */

package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestMultiBackendSupportExternalSecret(t *testing.T) {
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

			t.Logf("Creating dummy agent CR with external secret")
			err = decoder.ApplyWithManifestDir(ctx, r, "../config/samples", "instana_v1_instanaagent_multiple_backends_external_keyssecret.yaml", []resources.CreateOption{})
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("CR created")

			return ctx
		}).
		Assess("wait for first k8sensor deployment to become ready", WaitForDeploymentToBecomeReady(K8sensorDeploymentName)).
		Assess("wait for second k8sensor deployment to become ready", WaitForDeploymentToBecomeReady(fmt.Sprintf("%s-1", K8sensorDeploymentName))).
		Assess("wait for agent daemonset to become ready", WaitForAgentDaemonSetToBecomeReady()).
		Assess("validate instana-agent-config secret contains 2 backends", ValidateAgentMultiBackendConfiguration()).
		Feature()

	// test feature
	testEnv.Test(t, installCrWithExternalSecretFeature)
}

func TestMultiBackendSupportInlineSecret(t *testing.T) {
	installCrWithInlineSecretFeature := features.New("multiple backend support with inlined keyssecret").
		Setup(SetupOperatorDevBuild()).
		Setup(WaitForDeploymentToBecomeReady(InstanaOperatorDeploymentName)).
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}
			err = instanav1.AddToScheme(r.GetScheme())
			if err != nil {
				t.Fatal(err)
			}
			r.WithNamespace(cfg.Namespace())

			// read the same custom resource, but adjust it
			f, err := os.Open("../config/samples/instana_v1_instanaagent_multiple_backends.yaml")
			if err != nil {
				t.Fatal(err)
			}
			var agent instanav1.InstanaAgent
			err = decoder.Decode(f, &agent)
			if err != nil {
				t.Fatal("Could not decode agent", err)
			}

			t.Logf("Creating dummy agent CR with inline key")

			err = decoder.CreateHandler(r)(ctx, &agent)
			if err != nil {
				t.Fatal(err)
			}

			t.Logf("CR created")
			return ctx
		}).
		Assess("wait for k8sensor deployment to become ready", WaitForDeploymentToBecomeReady(K8sensorDeploymentName)).
		Assess("wait for second k8sensor deployment to become ready", WaitForDeploymentToBecomeReady(fmt.Sprintf("%s-1", K8sensorDeploymentName))).
		Assess("wait for agent daemonset to become ready", WaitForAgentDaemonSetToBecomeReady()).
		Assess("validate instana-agent-config secret contains 2 backends", ValidateAgentMultiBackendConfiguration()).
		Feature()

	// test feature
	testEnv.Test(t, installCrWithInlineSecretFeature)
}
