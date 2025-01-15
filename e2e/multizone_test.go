/*
 * (c) Copyright IBM Corp. 2024
 * (c) Copyright Instana Inc. 2024
 */

package e2e

import (
	"context"
	"testing"

	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestMultiZones(t *testing.T) {
	installWithMultiZones := features.New("multizone agent").
		Setup(SetupOperatorDevBuild()).
		Setup(WaitForDeploymentToBecomeReady(InstanaOperatorDeploymentName)).
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			t.Logf("Creating dummy secret")

			t.Logf("Creating dummy agent CR with extra volume")
			err = decoder.ApplyWithManifestDir(ctx, r, "../config/samples", "instana_v1_extended_instanaagent.yaml", []resources.CreateOption{})
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("CR created")

			return ctx
		}).
		Assess("wait for first k8sensor deployment to become ready", WaitForDeploymentToBecomeReady(K8sensorDeploymentName)).
		Assess("wait for agent daemonset in the first zone to become ready", WaitForAgentDaemonSetToBecomeReady("pool-01")).
		Assess("wait for agent daemonset in the second zone to become ready", WaitForAgentDaemonSetToBecomeReady("pool-02")).
		Feature()

	// test feature
	testEnv.Test(t, installWithMultiZones)
}
