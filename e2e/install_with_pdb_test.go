/*
 * (c) Copyright IBM Corp. 2025
 */

package e2e

import (
	"context"
	"testing"

	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	v1 "k8s.io/api/policy/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestInstallWithK8sensorPodDisruptionBudget(t *testing.T) {
	agent := NewAgentCr()
	enabled := true
	agent.Spec.K8sSensor.PodDisruptionBudget.Enabled = &enabled
	f := features.New("install dev-operator-build and enable k8sensor podDisruptionBudget").
		Setup(SetupOperatorDevBuild()).
		Setup(DeployAgentCr(&agent)).
		Assess("wait for instana-agent-controller-manager deployment to become ready", WaitForDeploymentToBecomeReady(InstanaOperatorDeploymentName)).
		Assess("wait for k8sensor deployment to become ready", WaitForDeploymentToBecomeReady(K8sensorDeploymentName)).
		Assess("check if instana-agent-controller-manager was able to deploy a podDisruptionBudget for the k8sensor", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal("Cleanup: Error initializing client", err)
			}
			r.WithNamespace(cfg.Namespace())
			pdb := &v1.PodDisruptionBudget{}
			h := helpers.NewHelpers(&agent)
			err = r.Get(ctx, h.K8sSensorResourcesName(), cfg.Namespace(), pdb)
			if err != nil {
				t.Fatal("Error fetching the pod disruption budget", err)
			}
			if pdb.Spec.MinAvailable.IntValue() != 2 {
				t.Fatal("The poddisruptionbudget found was not defining 2 MinAvailable instances", pdb.Spec.MinAvailable)
			}

			return ctx
		}).
		Feature()

	// test feature
	testEnv.Test(t, f)
}
