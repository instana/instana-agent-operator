/*
 * (c) Copyright IBM Corp. 2024
 * (c) Copyright Instana Inc. 2024
 */

package e2e

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestInitialInstall(t *testing.T) {
	namespace := "instana-agent"
	f1 := features.New("ensure-controller-manager-running").
		Assess("pods from "+namespace, func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := cfg.NewClient()
			if err != nil {
				t.Fatal(err)
			}
			dep := appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "controller-manager", Namespace: cfg.Namespace()},
			}
			// wait for operator pods of the deployment to become ready
			err = wait.For(conditions.New(client.Resources()).ResourceMatch(&dep, func(object k8s.Object) bool {
				return dep.Status.ReadyReplicas == *dep.Spec.Replicas
			}), wait.WithTimeout(time.Minute*2))

			if err != nil {
				t.Fatal(err)
			}
			t.Log("Deployment is ready")
			return ctx
		}).Feature()

	// test feature
	testEnv.Test(t, f1)
}
