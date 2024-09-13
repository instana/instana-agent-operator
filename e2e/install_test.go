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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

			p = utils.RunCommand("kubectl apply -f ../config/samples/instana_v1_instanaagent.yaml")
			if p.Err() != nil {
				t.Fatal("Error while applying example Agent CR", p.Command(), p.Err(), p.Out(), p.ExitCode())
			}

			return ctx
		}).
		Assess("wait for controller-manager deployment to become ready", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
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

			return ctx
		}).
		Assess("wait for k8sensor deployment to become ready", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := cfg.NewClient()
			if err != nil {
				t.Fatal(err)
			}

			dep := appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "instana-agent-k8sensor", Namespace: cfg.Namespace()},
			}
			// wait for operator pods of the deployment to become ready
			err = wait.For(conditions.New(client.Resources()).DeploymentConditionMatch(&dep, appsv1.DeploymentAvailable, corev1.ConditionTrue), wait.WithTimeout(time.Minute*2))
			if err != nil {
				t.Error(err)
			}

			return ctx
		}).
		Assess("wait for agent daemonset to become ready", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := cfg.NewClient()
			if err != nil {
				t.Fatal(err)
			}
			ds := appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{Name: "instana-agent", Namespace: cfg.Namespace()},
			}
			err = wait.For(conditions.New(client.Resources()).DaemonSetReady(&ds), wait.WithTimeout(time.Minute*2))
			if err != nil {
				t.Error(err)
			}

			return ctx
		}).
		Feature()

	// test feature
	testEnv.Test(t, f1)
}
