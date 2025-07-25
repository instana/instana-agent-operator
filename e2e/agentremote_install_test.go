/*
 * (c) Copyright IBM Corp. 2025
 * (c) Copyright Instana Inc. 2025
 */

package e2e

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestInitialRemoteInstall(t *testing.T) {
	agent := NewAgentRemoteCr(AgentRemoteCustomResourceName)
	initialInstallFeature := features.New("initial install dev-operator-build").
		Setup(SetupOperatorDevBuild()).
		Setup(DeployAgentRemoteCr(&agent)).
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
		Assess("wait for instana agent remote deployment to become ready", WaitForDeploymentToBecomeReady(AgentRemoteDeploymentName+AgentRemoteCustomResourceName)).
		Assess("check agent log for successful connection", WaitForAgentRemoteSuccessfulBackendConnection()).
		Feature()

	// test feature
	testEnv.Test(t, initialInstallFeature)
}
func TestUpdateRemoteInstall(t *testing.T) {
	agent := NewAgentRemoteCr(AgentRemoteCustomResourceName)
	// Cannot currently use as instana agent remote does not exist in latest version
	// installLatestFeature := features.New("deploy latest released instana-agent-operator").
	// 	Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	// 		const latestOperatorYamlUrl string = "https://github.com/instana/instana-agent-operator/releases/latest/download/instana-agent-operator.yaml"
	// 		t.Logf("Installing latest available operator from %s", latestOperatorYamlUrl)
	// 		p := utils.RunCommand(
	// 			fmt.Sprintf("kubectl apply -f %s", latestOperatorYamlUrl),
	// 		)
	// 		if p.Err() != nil {
	// 			t.Fatal("Error while applying latest operator yaml", p.Command(), p.Err(), p.Out(), p.ExitCode())
	// 		}
	// 		return ctx
	// 	}).
	// 	Setup(WaitForDeploymentToBecomeReady(InstanaOperatorDeploymentName)).
	// 	Setup(DeployAgentRemoteCr(&agent)).
	// 	Assess("wait for instana agent remote deployment to become ready", WaitForDeploymentToBecomeReady(AgentRemoteDeploymentName+AgentRemoteCustomResourceName)).
	// 	Assess("check agent log for successful connection", WaitForAgentRemoteSuccessfulBackendConnection()).
	// 	Feature()

	updateInstallDevBuildFeature := features.New("install dev-operator-build").
		Setup(SetupOperatorDevBuild()).
		Setup(DeployAgentRemoteCr(&agent)).
		Assess("wait for instana-agent-controller-manager deployment to become ready", WaitForDeploymentToBecomeReady(InstanaOperatorDeploymentName)).
		Assess("wait for instana remote agent deployment to become ready", WaitForDeploymentToBecomeReady(AgentRemoteDeploymentName+AgentRemoteCustomResourceName)).
		Assess("check agent log for successful connection", WaitForAgentRemoteSuccessfulBackendConnection()).
		Feature()

	checkReconciliationFeature := features.New("check reconcile works with new operator deployment").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			t.Log("Delete instana agent remote Deployment")
			var dep appsv1.Deployment
			if err := cfg.Client().Resources().Get(ctx, AgentRemoteDeploymentName+AgentRemoteCustomResourceName, cfg.Namespace(), &dep); err != nil {
				t.Fatal(err)
			}

			if err := cfg.Client().Resources().Delete(ctx, &dep); err != nil {
				t.Fatal(err)
			}
			t.Log("instana agent remote Deployment deleted")
			t.Log("Assessing reconciliation now")
			return ctx
		}).
		Assess("wait for instana agent remote deployment to become ready", WaitForDeploymentToBecomeReady(AgentRemoteDeploymentName+AgentRemoteCustomResourceName)).
		Assess("check agent log for successful connection", WaitForAgentRemoteSuccessfulBackendConnection()).
		Feature()

	// test feature
	testEnv.Test(t, updateInstallDevBuildFeature, checkReconciliationFeature)
}
