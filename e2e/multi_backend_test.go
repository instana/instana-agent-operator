/*
 * (c) Copyright IBM Corp. 2024, 2025
 * (c) Copyright Instana Inc. 2024, 2025
 */

package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestMultiBackendSupportExternalSecret(t *testing.T) {
	installCrWithExternalSecretFeature := features.New("multiple backend support with external keyssecret").
		Setup(SetupOperatorDevBuild()).
		Setup(WaitForDeploymentToBecomeReady(InstanaOperatorDeploymentName)).
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			t.Logf("Creating dummy secret")

			err = decoder.ApplyWithManifestDir(ctx, r, "../config/samples", "external_secret_instana_agent_key.yaml", []resources.CreateOption{})
			if err != nil {
				t.Fatal(err)
			}

			t.Logf("Secret created")

			t.Logf("Creating dummy agent CR with external secret")
			err = decoder.ApplyWithManifestDir(ctx, r, "../config/samples", "instana_v1_instanaagent_multiple_backends_external_keyssecret.yaml", []resources.CreateOption{})
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("CR created")

			return ctx
		}).
		Assess("wait for first k8sensor deployment to become ready", WaitForDeploymentToBecomeReady(K8sensorDeploymentName)).
		Assess("wait for second k8sensor deployment to become ready", WaitForDeploymentToBecomeReady(fmt.Sprintf("%s-1", K8sensorDeploymentName))).
		Assess("wait for agent daemonset to become ready", WaitForAgentDaemonSetToBecomeReady()).
		Assess("validate instana-agent-config secret contains 2 backends", ValidateAgentMultiBackendConfiguration()).
		Feature()

	// test feature
	testEnv.Test(t, installCrWithExternalSecretFeature)
}

func TestMultiBackendSupportInlineSecret(t *testing.T) {
	installCrWithInlineSecretFeature := features.New("multiple backend support with inlined keyssecret").
		Setup(SetupOperatorDevBuild()).
		Setup(WaitForDeploymentToBecomeReady(InstanaOperatorDeploymentName)).
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}
			err = instanav1.AddToScheme(r.GetScheme())
			if err != nil {
				t.Fatal(err)
			}
			r.WithNamespace(cfg.Namespace())

			// read the same custom resource, but adjust it
			f, err := os.Open("../config/samples/instana_v1_instanaagent_multiple_backends.yaml")
			if err != nil {
				t.Fatal(err)
			}
			var agent instanav1.InstanaAgent
			err = decoder.Decode(f, &agent)
			if err != nil {
				t.Fatal("Could not decode agent", err)
			}

			t.Logf("Creating dummy agent CR with inline key")

			err = decoder.CreateHandler(r)(ctx, &agent)
			if err != nil {
				t.Fatal(err)
			}

			t.Logf("CR created")
			return ctx
		}).
		Assess("wait for k8sensor deployment to become ready", WaitForDeploymentToBecomeReady(K8sensorDeploymentName)).
		Assess("wait for second k8sensor deployment to become ready", WaitForDeploymentToBecomeReady(fmt.Sprintf("%s-1", K8sensorDeploymentName))).
		Assess("wait for agent daemonset to become ready", WaitForAgentDaemonSetToBecomeReady()).
		Assess("validate instana-agent-config secret contains 2 backends", ValidateAgentMultiBackendConfiguration()).
		Feature()

	// test feature
	testEnv.Test(t, installCrWithInlineSecretFeature)
}

func TestRemovalOfAdditionalBackend(t *testing.T) {
	agent := NewAgentCr()

	agent.Spec.Agent.AdditionalBackends = append(agent.Spec.Agent.AdditionalBackends, instanav1.BackendSpec{
		EndpointHost: "test1.instana.ibm.com",
		EndpointPort: "443",
		Key:          "yyy",
	})

	agent.Spec.Agent.AdditionalBackends = append(agent.Spec.Agent.AdditionalBackends, instanav1.BackendSpec{
		EndpointHost: "test2.instana.ibm.com",
		EndpointPort: "443",
		Key:          "zzz",
	})

	var checksum, checksum1, checksum2 string

	installDevBuildWithTwoAdditionalBackendsFeature := features.New("install with 2 additional backends").
		Setup(SetupOperatorDevBuild()).
		Setup(DeployAgentCr(&agent)).
		Assess("wait for instana-agent-controller-manager deployment to become ready", WaitForDeploymentToBecomeReady(InstanaOperatorDeploymentName)).
		Assess("wait for k8sensor deployment to become ready", WaitForDeploymentToBecomeReady(K8sensorDeploymentName)).
		Assess("wait for k8sensor deployment to become ready", WaitForDeploymentToBecomeReady(K8sensorDeploymentName+"-1")).
		Assess("wait for k8sensor deployment to become ready", WaitForDeploymentToBecomeReady(K8sensorDeploymentName+"-2")).
		Assess("collect backend checksums", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			t.Logf("Collecting backend checksums from k8sensor deployments")
			var k8sensorDeployment, k8sensorDeployment1, k8sensorDeployment2 appsv1.Deployment
			if err := cfg.Client().Resources().Get(ctx, K8sensorDeploymentName, cfg.Namespace(), &k8sensorDeployment); err != nil {
				t.Fatal(err)
			}

			if err := cfg.Client().Resources().Get(ctx, K8sensorDeploymentName+"-1", cfg.Namespace(), &k8sensorDeployment1); err != nil {
				t.Fatal(err)
			}

			if err := cfg.Client().Resources().Get(ctx, K8sensorDeploymentName+"-2", cfg.Namespace(), &k8sensorDeployment2); err != nil {
				t.Fatal(err)
			}

			checksum = k8sensorDeployment.Spec.Template.ObjectMeta.Annotations["checksum/backend"]
			checksum1 = k8sensorDeployment1.Spec.Template.ObjectMeta.Annotations["checksum/backend"]
			checksum2 = k8sensorDeployment2.Spec.Template.ObjectMeta.Annotations["checksum/backend"]

			t.Logf("k8sensor deployment: %s", checksum)
			t.Logf("k8sensor deployment-1: %s", checksum1)
			t.Logf("k8sensor deployment-2: %s", checksum2)
			return ctx
		}).
		Assess("wait for agent daemonset to become ready", WaitForAgentDaemonSetToBecomeReady()).
		Feature()

	agent2 := NewAgentCr()
	agent2.Spec.Agent.AdditionalBackends = []instanav1.BackendSpec{}
	agent2.Spec.Agent.AdditionalBackends = append(agent2.Spec.Agent.AdditionalBackends, instanav1.BackendSpec{
		EndpointHost: "test2.instana.ibm.com",
		EndpointPort: "443",
		Key:          "zzz",
	})
	checkReconciliationFeature := features.New("check reconcile works with new operator deployment").
		Setup(UpdateAgentCr(&agent2)).
		Assess("wait for k8sensor deployment to become ready", WaitForDeploymentToBecomeReady(K8sensorDeploymentName)).
		Assess("wait for k8sensor deployment to become ready", WaitForDeploymentToBecomeReady(K8sensorDeploymentName+"-1")).
		Assess("wait for agent daemonset to become ready", WaitForAgentDaemonSetToBecomeReady()).
		Assess("ensure old k8sensor deployment 2 is deleted", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			k8sensorDeploymentName2 := K8sensorDeploymentName + "-2"
			t.Logf("Ensuring the old deployment %s is not running", k8sensorDeploymentName2)
			client, err := cfg.NewClient()
			if err != nil {
				t.Fatal(err)
			}
			dep := appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: k8sensorDeploymentName2, Namespace: cfg.Namespace()},
			}
			err = wait.For(conditions.New(client.Resources()).ResourceDeleted(&dep), wait.WithTimeout(time.Minute*2))
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("Deployment %s is deleted", k8sensorDeploymentName2)
			return ctx
		}).
		Assess("ensure backend checksums changed correctly", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			t.Logf("Collecting backend checksums from k8sensor deployments")
			var k8sensorDeployment, k8sensorDeployment1 appsv1.Deployment
			if err := cfg.Client().Resources().Get(ctx, K8sensorDeploymentName, cfg.Namespace(), &k8sensorDeployment); err != nil {
				t.Fatal(err)
			}

			if err := cfg.Client().Resources().Get(ctx, K8sensorDeploymentName+"-1", cfg.Namespace(), &k8sensorDeployment1); err != nil {
				t.Fatal(err)
			}

			newChecksum := k8sensorDeployment.Spec.Template.ObjectMeta.Annotations["checksum/backend"]
			newChecksum1 := k8sensorDeployment1.Spec.Template.ObjectMeta.Annotations["checksum/backend"]

			t.Logf("k8sensor deployment: %s", newChecksum)
			t.Logf("k8sensor deployment-1: %s", newChecksum1)

			if newChecksum != checksum {
				t.Errorf("If the additional backend 1 gets removed, the main backend should remain unchanged")
			}

			if newChecksum1 != checksum2 {
				t.Errorf("If additional backend 1 gets removed, k8sensor deployment 1 must carry the checksum of the former backend2")
			}
			return ctx
		}).
		Feature()

	// test feature
	testEnv.Test(t, installDevBuildWithTwoAdditionalBackendsFeature, checkReconciliationFeature)
}
