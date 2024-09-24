/*
 * (c) Copyright IBM Corp. 2024
 * (c) Copyright Instana Inc. 2024
 */

package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	v1 "github.com/instana/instana-agent-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/support/utils"
)

func TestUpdateInstall(t *testing.T) {
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

			// Wait for controller-manager deployment to ensure that CRD is installed correctly before proceeding.
			// Technically, it could be categorized as "Assess" method, but the setup process requires to wait in between.
			// Therefore, keeping the wait logic in this section.
			client, err := cfg.NewClient()
			if err != nil {
				t.Fatal(err)
			}
			dep := appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: InstanaOperatorDeploymentName, Namespace: cfg.Namespace()},
			}

			t.Log("Waiting for operator deployment to become ready")

			err = wait.For(conditions.New(client.Resources()).DeploymentConditionMatch(&dep, appsv1.DeploymentAvailable, corev1.ConditionTrue), wait.WithTimeout(time.Minute*2))
			if err != nil {
				t.Fatal(err)
			}

			t.Log("Creating a new Agent CR")

			// Create Agent CR
			agent := NewAgentCr(t)
			r := client.Resources(cfg.Namespace())
			err = v1.AddToScheme(r.GetScheme())
			if err != nil {
				t.Fatal("Could not add Agent CR to client scheme", err)
			}

			err = r.Create(ctx, &agent)
			if err != nil {
				t.Fatal("Could not create Agent CR", err)
			}

			return ctx
		}).
		Assess("wait for k8sensor deployment to become ready", WaitForDeploymentToBecomeReady("instana-agent-k8sensor")).
		Assess("wait for agent daemonset to become ready", WaitForAgentDaemonSetToBecomeReady()).
		Assess("check agent log for successful connection", WaitForAgentSuccessfulBackendConnection()).
		Feature()

	updateInstallDevBuildFeature := features.New("upgrade install from latest released to dev-operator-build").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// Create pull secret for custom registry
			t.Logf("Creating custom pull secret for %s", InstanaTestCfg.ContainerRegistry.Host)
			p := utils.RunCommand(
				fmt.Sprintf("kubectl create secret -n %s docker-registry %s --docker-server=%s --docker-username=%s --docker-password=%s",
					cfg.Namespace(),
					InstanaTestCfg.ContainerRegistry.Name,
					InstanaTestCfg.ContainerRegistry.Host,
					InstanaTestCfg.ContainerRegistry.User,
					InstanaTestCfg.ContainerRegistry.Password),
			)
			if p.Err() != nil {
				t.Fatal("Error while creating pull secret", p.Command(), p.Err(), p.Out(), p.ExitCode())
			}
			t.Log("Pull secret created")

			// Use make logic to ensure that local dev commands and test commands are in sync
			t.Log("Deploy new dev build by running: make install deploy")
			p = utils.RunCommand(fmt.Sprintf("bash -c 'cd .. && IMG=%s:%s make install deploy'", InstanaTestCfg.OperatorImage.Name, InstanaTestCfg.OperatorImage.Tag))
			if p.Err() != nil {
				t.Fatal("Error while deploying custom operator build during update installation", p.Command(), p.Err(), p.Out(), p.ExitCode())
			}
			t.Log("Deployment submitted")

			// Inject image pull secret into deployment, ensure to scale to 0 replicas and back to 2 replicas, otherwise pull secrets are not propagated correctly
			t.Log("Patch instana operator deployment to redeploy pods with image pull secret")
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal("Cleanup: Error initializing client", err)
			}
			r.WithNamespace(cfg.Namespace())
			agent := &appsv1.Deployment{}
			err = r.Get(ctx, InstanaOperatorDeploymentName, cfg.Namespace(), agent)
			if err != nil {
				t.Fatal("Failed to get deployment-manager deployment", err)
			}
			err = r.Patch(ctx, agent, k8s.Patch{
				PatchType: types.MergePatchType,
				Data:      []byte(fmt.Sprintf(`{"spec":{ "replicas": 0, "template":{"spec": {"imagePullSecrets": [{"name": "%s"}]}}}}`, InstanaTestCfg.ContainerRegistry.Name)),
			})
			if err != nil {
				t.Fatal("Failed to patch deployment to include pull secret and 0 replicas", err)
			}

			err = r.Patch(ctx, agent, k8s.Patch{
				PatchType: types.MergePatchType,
				Data:      []byte(`{"spec":{ "replicas": 2 }}`),
			})
			if err != nil {
				t.Fatal("Failed to patch deployment to include pull secret and 0 replicas", err)
			}
			t.Log("Patching completed")
			return ctx
		}).
		Assess("wait for controller-manager deployment to become ready", WaitForDeploymentToBecomeReady(InstanaOperatorDeploymentName)).
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
