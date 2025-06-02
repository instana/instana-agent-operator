/*
 * (c) Copyright IBM Corp. 2024
 * (c) Copyright Instana Inc. 2024
 */

package e2e

import (
	"context"
	"fmt"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/support/utils"
)

func TestInitialInstall(t *testing.T) {
	agent := NewAgentCr()
	initialInstallFeature := features.New("initial install dev-operator-build").
		Setup(SetupOperatorDevBuild()).
		Setup(DeployAgentCr(&agent)).
		Assess("wait for instana-agent-controller-manager deployment to become ready", WaitForDeploymentToBecomeReady(InstanaOperatorDeploymentName)).
		Assess("check for single instance of instana-agent-controller-manager", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal("Cleanup: Error initializing client", err)
			}
			r.WithNamespace(cfg.Namespace())
			agent := &appsv1.Deployment{}
			err = r.Get(ctx, InstanaOperatorDeploymentName, cfg.Namespace(), agent)
			if err != nil {
				t.Fatal("Cleanup: Error fetching the operator deployment", err)
			}

			expectedReplicas := new(int32)
			*expectedReplicas = 1
			if *agent.Spec.Replicas != *expectedReplicas {
				t.Fatal("Unexpected number of replicas", *agent.Spec.Replicas, *expectedReplicas)
			}
			return ctx
		}).
		Assess("wait for k8sensor deployment to become ready", WaitForDeploymentToBecomeReady(K8sensorDeploymentName)).
		Assess("wait for agent daemonset to become ready", WaitForAgentDaemonSetToBecomeReady()).
		Assess("check agent log for successful connection", WaitForAgentSuccessfulBackendConnection()).
		Feature()

	// test feature
	testEnv.Test(t, initialInstallFeature)
}
func TestUpdateInstall(t *testing.T) {
	agent := NewAgentCr()
	installLatestFeature := features.New("deploy latest released instana-agent-operator").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			const latestOperatorYamlUrl string = "https://github.com/instana/instana-agent-operator/releases/latest/download/instana-agent-operator.yaml"
			t.Logf("Installing latest available operator from %s", latestOperatorYamlUrl)
			p := utils.RunCommand(
				fmt.Sprintf("kubectl apply -f %s", latestOperatorYamlUrl),
			)
			if p.Err() != nil {
				t.Fatal("Error while applying latest operator yaml", p.Command(), p.Err(), p.Out(), p.ExitCode())
			}
			return ctx
		}).
		Setup(WaitForDeploymentToBecomeReady(InstanaOperatorDeploymentName)).
		Setup(DeployAgentCr(&agent)).
		Assess("wait for k8sensor deployment to become ready", WaitForDeploymentToBecomeReady(K8sensorDeploymentName)).
		Assess("wait for agent daemonset to become ready", WaitForAgentDaemonSetToBecomeReady()).
		Assess("check agent log for successful connection", WaitForAgentSuccessfulBackendConnection()).
		Feature()

	updateInstallDevBuildFeature := features.New("upgrade install from latest released to dev-operator-build").
		Setup(SetupOperatorDevBuild()).
		Assess("wait for instana-agent-controller-manager deployment to become ready", WaitForDeploymentToBecomeReady(InstanaOperatorDeploymentName)).
		Assess("wait for k8sensor deployment to become ready", WaitForDeploymentToBecomeReady(K8sensorDeploymentName)).
		Assess("wait for agent daemonset to become ready", WaitForAgentDaemonSetToBecomeReady()).
		Assess("check agent log for successful connection", WaitForAgentSuccessfulBackendConnection()).
		Feature()

	checkReconciliationFeature := features.New("check reconcile works with new operator deployment").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// delete agent daemonset
			t.Log("Delete agent DaemonSet")
			var ds appsv1.DaemonSet
			if err := cfg.Client().Resources().Get(ctx, AgentDaemonSetName, cfg.Namespace(), &ds); err != nil {
				t.Fatal(err)
			}
			if err := cfg.Client().Resources().Delete(ctx, &ds); err != nil {
				t.Fatal(err)
			}
			t.Log("Agent DaemonSet deleted")

			t.Log("Delete k8sensor Deployment")
			var dep appsv1.Deployment
			if err := cfg.Client().Resources().Get(ctx, K8sensorDeploymentName, cfg.Namespace(), &dep); err != nil {
				t.Fatal(err)
			}

			if err := cfg.Client().Resources().Delete(ctx, &dep); err != nil {
				t.Fatal(err)
			}
			t.Log("K8sensor Deployment deleted")
			t.Log("Assessing reconciliation now")
			return ctx
		}).
		Assess("wait for k8sensor deployment to become ready", WaitForDeploymentToBecomeReady("instana-agent-k8sensor")).
		Assess("wait for agent daemonset to become ready", WaitForAgentDaemonSetToBecomeReady()).
		Assess("check agent log for successful connection", WaitForAgentSuccessfulBackendConnection()).
		Feature()

	// test feature
	testEnv.Test(t, installLatestFeature, updateInstallDevBuildFeature, checkReconciliationFeature)
}
