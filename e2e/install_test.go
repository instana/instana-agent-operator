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
	"strings"
	"testing"
	"time"

	v1 "github.com/instana/instana-agent-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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

			// Wait for controller-manager deployment to ensure that CRD is installed correctly before proceeding
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

			// using kubectl to apply real yaml file
			// p = utils.RunCommand("kubectl apply -f ../config/samples/instana_v1_instanaagent.yaml")
			// if p.Err() != nil {
			// 	t.Fatal("Error while applying example Agent CR", p.Command(), p.Err(), p.Out(), p.ExitCode())
			// }

			// using API to create Agent CR
			agent := NewAgentCr(t)
			r := client.Resources(namespace)
			v1.AddToScheme(r.GetScheme())
			err = r.Create(ctx, &agent)
			if err != nil {
				t.Fatal("Could not create Agent CR", err)
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
		Assess("check agent log for successful connection", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			clientSet, err := kubernetes.NewForConfig(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}
			podList, err := clientSet.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: "app.kubernetes.io/component=instana-agent"})
			if err != nil {
				t.Fatal(err)
			}
			if len(podList.Items) == 0 {
				t.Fatal("No pods found")
			}

			connectionSuccessful := false
			var buf *bytes.Buffer
			for i := 0; i < 6; i++ {
				time.Sleep(10 * time.Second)
				logReq := clientSet.CoreV1().Pods(namespace).GetLogs(podList.Items[0].Name, &corev1.PodLogOptions{})
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
				t.Error("Agent pod did not log successful connection, dumping log", buf.String())
			}

			return ctx
		}).
		Feature()

	// test feature
	testEnv.Test(t, f1)
}
