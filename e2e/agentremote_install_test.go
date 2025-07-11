/*
 * (c) Copyright IBM Corp. 2025
 * (c) Copyright Instana Inc. 2025
 */

package e2e

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	v1 "github.com/instana/instana-agent-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	e2etypes "sigs.k8s.io/e2e-framework/pkg/types"
	"sigs.k8s.io/e2e-framework/support/utils"
)

var (
	AgentRemoteDeploymentName = "instana-agent-r-"
)

func TestInitialRemoteInstall(t *testing.T) {
	agent := NewAgentRemoteCr("remote-1")
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
		Assess("wait for instana agent remote deployment to become ready", WaitForDeploymentToBecomeReady(AgentRemoteDeploymentName+"remote-1")).
		Assess("check agent log for successful connection", WaitForAgentRemoteSuccessfulBackendConnection()).
		Feature()

	// test feature
	testEnv.Test(t, initialInstallFeature)
}
func TestUpdateRemoteInstall(t *testing.T) {
	agent := NewAgentRemoteCr("remote-1")
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
		Setup(DeployAgentRemoteCr(&agent)).
		Assess("wait for instana agent remote deployment to become ready", WaitForDeploymentToBecomeReady(AgentRemoteDeploymentName+"remote-1")).
		Assess("check agent log for successful connection", WaitForAgentRemoteSuccessfulBackendConnection()).
		Feature()

	updateInstallDevBuildFeature := features.New("upgrade install from latest released to dev-operator-build").
		Setup(SetupOperatorDevBuild()).
		Assess("wait for instana-agent-controller-manager deployment to become ready", WaitForDeploymentToBecomeReady(InstanaOperatorDeploymentName)).
		Assess("wait for instana remote agent deployment to become ready", WaitForDeploymentToBecomeReady(AgentRemoteDeploymentName+"remote-1")).
		Assess("check agent log for successful connection", WaitForAgentRemoteSuccessfulBackendConnection()).
		Feature()

	checkReconciliationFeature := features.New("check reconcile works with new operator deployment").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			t.Log("Delete instana agent remote Deployment")
			var dep appsv1.Deployment
			if err := cfg.Client().Resources().Get(ctx, AgentRemoteDeploymentName+"remote-1", cfg.Namespace(), &dep); err != nil {
				t.Fatal(err)
			}

			if err := cfg.Client().Resources().Delete(ctx, &dep); err != nil {
				t.Fatal(err)
			}
			t.Log("instana agent remote Deployment deleted")
			t.Log("Assessing reconciliation now")
			return ctx
		}).
		Assess("wait for instana agent remote deployment to become ready", WaitForDeploymentToBecomeReady(AgentRemoteDeploymentName+"remote-1")).
		Assess("check agent log for successful connection", WaitForAgentRemoteSuccessfulBackendConnection()).
		Feature()

	// test feature
	testEnv.Test(t, installLatestFeature, updateInstallDevBuildFeature, checkReconciliationFeature)
}

func NewAgentRemoteCr(name string) v1.InstanaAgentRemote {

	return v1.InstanaAgentRemote{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: InstanaNamespace,
		},
		Spec: v1.InstanaAgentRemoteSpec{
			Zone: v1.Name{
				Name: "e2e",
			},
			// ensure to not overlap between concurrent test runs on different clusters, randomize cluster name, but have consistent zone
			ConfigurationYaml: "testing",
		},
	}
}

func DeployAgentRemoteCr(agent *v1.InstanaAgentRemote) e2etypes.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
		}
		t.Log("Creating a new Agent Remote CR")

		// Create Agent CR
		r := client.Resources(cfg.Namespace())
		err = v1.AddToScheme(r.GetScheme())
		if err != nil {
			t.Fatal("Could not add Agent CR to client scheme", err)
		}

		err = r.Create(ctx, agent)
		if err != nil {
			t.Fatal("Could not create Agent Remote CR", err)
		}

		return ctx
	}
}

func WaitForAgentRemoteSuccessfulBackendConnection() e2etypes.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Log("Searching for successful backend connection in agent logs")
		clientSet, err := kubernetes.NewForConfig(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(20 * time.Second)
		podList, err := clientSet.CoreV1().Pods(cfg.Namespace()).List(ctx, metav1.ListOptions{LabelSelector: "app.kubernetes.io/component=instana-agent-remote"})
		if err != nil {
			t.Fatal(err)
		}
		if len(podList.Items) == 0 {
			t.Fatal("No pods found")
		}

		connectionSuccessful := false
		var buf *bytes.Buffer
		for i := 0; i < 9; i++ {
			t.Log("Sleeping 20 seconds")
			time.Sleep(20 * time.Second)
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
