/*
 * (c) Copyright IBM Corp. 2025
 * (c) Copyright Instana Inc. 2025
 */

package e2e

import (
	"context"
	"fmt"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/support/utils"
)

func TestUpdateInstallFromOldGenericResourceNames(t *testing.T) {
	agent := NewAgentCr()
	installLatestFeature := features.New("deploy instana-agent-operator with the generic resource names (controller-manager, manager-role and manager-rolebinding)").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			const oldResourceNamesOperatorYamlUrl string = "https://github.com/instana/instana-agent-operator/releases/download/v2.1.14/instana-agent-operator.yaml"
			t.Logf("Installing latest operator with the old, generic resource names from %s", oldResourceNamesOperatorYamlUrl)
			p := utils.RunCommand(
				fmt.Sprintf("kubectl apply -f %s", oldResourceNamesOperatorYamlUrl),
			)
			if p.Err() != nil {
				t.Fatal("Error while applying the old operator yaml", p.Command(), p.Err(), p.Out(), p.ExitCode())
			}
			return ctx
		}).
		Setup(WaitForDeploymentToBecomeReady(InstanaOperatorOldDeploymentName)).
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
		Assess("confirm the old deployment is gone", EnsureOldControllerManagerDeploymentIsNotRunning()).
		Assess("confirm the old clusterrole is gone", EnsureOldClusterRoleIsGone()).
		Assess("confirm the old clusterrolebinding is gone", EnsureOldClusterRoleBindingIsGone()).
		Assess("wait for k8sensor deployment to become ready", WaitForDeploymentToBecomeReady("instana-agent-k8sensor")).
		Assess("wait for agent daemonset to become ready", WaitForAgentDaemonSetToBecomeReady()).
		Assess("check agent log for successful connection", WaitForAgentSuccessfulBackendConnection()).
		Feature()

	// test feature
	testEnv.Test(t, installLatestFeature, updateInstallDevBuildFeature, checkReconciliationFeature)
}
