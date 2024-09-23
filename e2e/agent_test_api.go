/*
 * (c) Copyright IBM Corp. 2024
 * (c) Copyright Instana Inc. 2024
 */

package e2e

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

func WaitForDeploymentToBecomeReady(name string) func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
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
		return ctx
	}
}

func WaitForAgentDaemonSetToBecomeReady() func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
		}
		ds := appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{Name: "instana-agent", Namespace: cfg.Namespace()},
		}
		err = wait.For(conditions.New(client.Resources()).DaemonSetReady(&ds), wait.WithTimeout(time.Minute*5))
		if err != nil {
			t.Fatal(err)
		}
		return ctx
	}
}

func WaitForAgentSuccessfulBackendConnection() func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
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
			time.Sleep(10 * time.Second)
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
