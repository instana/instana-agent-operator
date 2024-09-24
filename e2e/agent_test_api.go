/*
 * (c) Copyright IBM Corp. 2024
 * (c) Copyright Instana Inc. 2024
 */

package e2e

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

	v1 "github.com/instana/instana-agent-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	e2etypes "sigs.k8s.io/e2e-framework/pkg/types"
	"sigs.k8s.io/e2e-framework/support/utils"
)

// This file exposes the reusable assets which are used during the e2e test

// Setup functions
func SetupOperatorDevBuild() e2etypes.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
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
	}
}

// Assess functions
func WaitForDeploymentToBecomeReady(name string) e2etypes.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Logf("Waiting for deployment %s to become ready", name)
		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
		}
		dep := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: cfg.Namespace()},
		}
		// wait for operator pods of the deployment to become ready
		err = wait.For(conditions.New(client.Resources()).DeploymentConditionMatch(&dep, appsv1.DeploymentAvailable, corev1.ConditionTrue), wait.WithTimeout(time.Minute*2))
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Deployment %s is ready", name)
		return ctx
	}
}

func WaitForAgentDaemonSetToBecomeReady() e2etypes.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Logf("Waiting for DaemonSet %s is ready", AgentDaemonSetName)
		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
		}
		ds := appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{Name: AgentDaemonSetName, Namespace: cfg.Namespace()},
		}
		err = wait.For(conditions.New(client.Resources()).DaemonSetReady(&ds), wait.WithTimeout(time.Minute*5))
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("DaemonSet %s is ready", AgentDaemonSetName)
		return ctx
	}
}

func WaitForAgentSuccessfulBackendConnection() e2etypes.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Log("Searching for successful backend connection in agent logs")
		clientSet, err := kubernetes.NewForConfig(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}
		podList, err := clientSet.CoreV1().Pods(cfg.Namespace()).List(ctx, metav1.ListOptions{LabelSelector: "app.kubernetes.io/component=instana-agent"})
		if err != nil {
			t.Fatal(err)
		}
		if len(podList.Items) == 0 {
			t.Fatal("No pods found")
		}

		connectionSuccessful := false
		var buf *bytes.Buffer
		for i := 0; i < 9; i++ {
			t.Log("Sleeping 10 seconds")
			time.Sleep(10 * time.Second)
			t.Log("Fetching logs")
			logReq := clientSet.CoreV1().Pods(cfg.Namespace()).GetLogs(podList.Items[0].Name, &corev1.PodLogOptions{})
			podLogs, err := logReq.Stream(ctx)
			if err != nil {
				t.Fatal("Could not stream logs", err)
			}
			defer podLogs.Close()

			buf = new(bytes.Buffer)
			_, err = io.Copy(buf, podLogs)

			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(buf.String(), "Connected using HTTP/2 to") {
				t.Log("Connection established correctly")
				connectionSuccessful = true
				break
			} else {
				t.Log("Could not find working connection in log of the first pod yet")
			}
		}
		if !connectionSuccessful {
			t.Fatal("Agent pod did not log successful connection, dumping log", buf.String())
		}
		return ctx
	}
}

// Helper to produce test structs
func NewAgentCr(t *testing.T) v1.InstanaAgent {
	boolTrue := true

	return v1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "instana-agent",
			Namespace: InstanaNamespace,
		},
		Spec: v1.InstanaAgentSpec{
			Zone: v1.Name{
				Name: "e2e",
			},
			// ensure to not overlap between concurrent test runs on different clusters, randomize cluster name, but have consistent zone
			Cluster: v1.Name{Name: envconf.RandomName("e2e", 4)},
			Agent: v1.BaseAgentSpec{
				Key:          InstanaTestCfg.InstanaBackend.AgentKey,
				EndpointHost: InstanaTestCfg.InstanaBackend.EndpointHost,
				EndpointPort: strconv.Itoa(InstanaTestCfg.InstanaBackend.EndpointPort),
			},
			OpenTelemetry: v1.OpenTelemetry{
				GRPC: &v1.Enabled{Enabled: &boolTrue},
				HTTP: &v1.Enabled{Enabled: &boolTrue},
			},
		},
	}
}
