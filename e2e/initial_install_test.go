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

	"github.com/stretchr/testify/assert"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/support/utils"
)

func TestInitialInstall(t *testing.T) {
	f1 := features.New("deploy instana-agent-operator").
		Assess("deploy latest released version", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			t.Log("Deploy latest yaml from GitHub")
			const latestOperatorYaml string = "https://github.com/instana/instana-agent-operator/releases/latest/download/instana-agent-operator.yaml"
			if p := utils.RunCommand(
				fmt.Sprintf("kubectl apply -f %s --server-side", latestOperatorYaml),
			); p.Err() != nil {
				assert.Nil(t, p.Err(), "Error while applying latest operator yaml")
			}

			client, err := cfg.NewClient()
			if err != nil {
				assert.Nil(t, err, "Could not create new client")
			}
			t.Log("Wait for controller manager deployment to become ready")
			dep := appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "controller-manager", Namespace: cfg.Namespace()},
			}
			// wait for operator pods of the deployment to become ready
			err = wait.For(conditions.New(client.Resources()).ResourceMatch(&dep, func(object k8s.Object) bool {
				return dep.Status.ReadyReplicas == *dep.Spec.Replicas
			}), wait.WithTimeout(time.Minute*2))

			if err != nil {
				assert.Nil(t, err)
			}
			t.Log("Deployment is ready")
			return ctx
		}).Feature()

	f2 := features.New("deploy agent cr").
		Assess("deploy example agent cr with defaults", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := cfg.NewClient()
			if err != nil {
				assert.Nil(t, err, "Could not create new client")
			}

			if p := utils.RunCommand("kubectl apply -f ../config/samples/instana_v1_instanaagent.yaml"); p.Err() != nil {
				assert.Nil(t, p.Err(), "Error while applying latest operator yaml")
				assert.Equal(t, 0, p.ExitCode())
			}

			t.Log("Wait for agent daemonset to become ready")
			ds := appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{Name: "instana-agent", Namespace: cfg.Namespace()},
			}
			err = wait.For(conditions.New(client.Resources()).DaemonSetReady(&ds), wait.WithTimeout(time.Minute*2))
			if err != nil {
				assert.Nil(t, err)
			}
			t.Log("Agent DeamonSet is ready")

			t.Log("Wait for k8sensor deployment to become ready")
			dep := appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "instana-agent-k8sensor", Namespace: cfg.Namespace()},
			}
			// wait for operator pods of the deployment to become ready
			err = wait.For(conditions.New(client.Resources()).ResourceMatch(&dep, func(object k8s.Object) bool {
				return dep.Status.ReadyReplicas == *dep.Spec.Replicas
			}), wait.WithTimeout(time.Minute*2))

			if err != nil {
				assert.Nil(t, err)
			}
			t.Log("K8sensor deployment is ready")
			return ctx
		}).Feature()

	// test feature
	testEnv.Test(t, f1, f2)
}