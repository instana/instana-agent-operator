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
	f1 := features.New("deploy latest released instana-agent-operator").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			const latestOperatorYaml string = "https://github.com/instana/instana-agent-operator/releases/latest/download/instana-agent-operator.yaml"
			p := utils.RunCommand(
				fmt.Sprintf("kubectl apply -f %s", latestOperatorYaml),
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
				ObjectMeta: metav1.ObjectMeta{Name: "controller-manager", Namespace: cfg.Namespace()},
			}

			err = wait.For(conditions.New(client.Resources()).DeploymentConditionMatch(&dep, appsv1.DeploymentAvailable, corev1.ConditionTrue), wait.WithTimeout(time.Minute*2))
			if err != nil {
				t.Fatal(err)
			}

			// Create Agent CR
			agent := NewAgentCr(t)
			r := client.Resources(namespace)
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

	f2 := features.New("upgrade install from latest released to dev-operator-build").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// Create pull secret for custom registry
			p := utils.RunCommand(
				fmt.Sprintf("kubectl create secret -n %s docker-registry delivery.instana --docker-server=%s --docker-username=%s --docker-password=%s",
					cfg.Namespace(),
					instanaTestConfig.ContainerRegistry.Host,
					instanaTestConfig.ContainerRegistry.User,
					instanaTestConfig.ContainerRegistry.Password),
			)
			if p.Err() != nil {
				t.Fatal("Error while creating pull secret", p.Command(), p.Err(), p.Out(), p.ExitCode())
			}

			// Use make logic to ensure that local dev commands and test commands are in sync
			p = utils.RunCommand(fmt.Sprintf("bash -c 'cd .. && IMG=%s:%s make install deploy'", instanaTestConfig.OperatorImage.Name, instanaTestConfig.OperatorImage.Tag))
			if p.Err() != nil {
				t.Fatal("Error while deploying custom operator build during update installation", p.Command(), p.Err(), p.Out(), p.ExitCode())
			}

			// Inject image pull secret into deployment, ensure to scale to 0 replicas and back to 2 replicas, otherwise pull secrets are not propagated correctly
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal("Cleanup: Error initializing client", err)
			}
			r.WithNamespace(namespace)
			agent := &appsv1.Deployment{}
			err = r.Get(ctx, "controller-manager", namespace, agent)
			if err != nil {
				t.Fatal("Failed to get deployment-manager deployment", err)
			}
			err = r.Patch(ctx, agent, k8s.Patch{
				PatchType: types.MergePatchType,
				Data:      []byte(`{"spec":{ "replicas": 0, "template":{"spec": {"imagePullSecrets": [{"name": "delivery.instana"}]}}}}`),
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

			// // delete existing pods to ensure the new pull secret is being used correctly
			// clientSet, err := kubernetes.NewForConfig(cfg.Client().RESTConfig())
			// if err != nil {
			// 	t.Error(err)
			// }

			// // podList, err := clientSet.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: "app.kubernetes.io/name=instana-agent-operator"})
			// // if err != nil {
			// // 	t.Error(err)
			// // }

			// clientSet.CoreV1().Pods(namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: "app.kubernetes.io/name=instana-agent-operator"})
			// if err != nil {
			// 	t.Error(err)
			// }

			return ctx
		}).
		Assess("wait for controller-manager deployment to become ready", WaitForDeploymentToBecomeReady("controller-manager")).
		Assess("wait for k8sensor deployment to become ready", WaitForDeploymentToBecomeReady("instana-agent-k8sensor")).
		Assess("wait for agent daemonset to become ready", WaitForAgentDaemonSetToBecomeReady()).
		Assess("check agent log for successful connection", WaitForAgentSuccessfulBackendConnection()).
		Feature()

	// test feature
	testEnv.Test(t, f1, f2)
}
