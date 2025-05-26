/*
 * (c) Copyright IBM Corp. 2025
 */

package e2e

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	log "k8s.io/klog/v2"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/support/utils"
)

func TestNamespaceLabelConfigmap(t *testing.T) {
	agent := NewAgentCr(t)
	installAndCheckNamespaceLabels := features.New("check namespace configmap in agent pods").
		Setup(SetupOperatorDevBuild()).
		Setup(DeployAgentCr(&agent)).
		Assess("wait for instana-agent-controller-manager deployment to become ready", WaitForDeploymentToBecomeReady(InstanaOperatorDeploymentName)).
		Assess("wait for k8sensor deployment to become ready", WaitForDeploymentToBecomeReady(K8sensorDeploymentName)).
		Assess("wait for agent daemonset to become ready", WaitForAgentDaemonSetToBecomeReady()).
		Assess("check agent pod for namespaces label file", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			log.Infof("Validating namespace labels")
			// Create a client to interact with the Kube API
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			pods := &corev1.PodList{}
			listOps := resources.WithLabelSelector("app.kubernetes.io/component=instana-agent")
			err = r.List(ctx, pods, listOps)
			if err != nil || pods.Items == nil {
				t.Error("error while getting pods", err)
			}
			var stdout, stderr bytes.Buffer
			podName := pods.Items[0].Name
			containerName := "instana-agent"

			if err := r.ExecInPod(
				ctx,
				cfg.Namespace(),
				podName,
				containerName,
				[]string{"bash", "-c", "cat ${NAMESPACES_DETAILS_PATH} | grep -A 5 'instana-agent:'"},
				&stdout,
				&stderr,
			); err != nil {
				t.Log(stderr.String())
				t.Error(err)
			}
			stringToMatch := "kubernetes.io/metadata.name"
			if strings.Contains(stdout.String(), stringToMatch) {
				t.Logf("ExecInPod returned expected namespace file")
			} else {
				t.Error(fmt.Sprintf("Expected to find %s in namespace file", stringToMatch), stdout.String())
			}
			return ctx
		}).
		Assess("check if configmap gets updated on new namespaces, updated labels and deleted namespaces", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			log.Infof("Validating new namespace are found in configmap")
			// Create a client to interact with the Kube API
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			// check that new namespace with random values is not present in the configmap yet
			yamlName := "namespaces.yaml"
			cm := &corev1.ConfigMap{}
			newNamespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: envconf.RandomName("operator-e2e-ns", 20),
					Labels: map[string]string{
						"agentOperatorTest": envconf.RandomName("operator-e2e-label", 30),
					},
				},
			}
			err = r.Get(ctx, "instana-agent-namespaces", cfg.Namespace(), cm)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(cm.Data[yamlName], newNamespace.ObjectMeta.Name+":") {
				t.Errorf("The namespace %s of the e2e test should not be present yet", newNamespace)
			}

			// Create the new namespace with random name
			t.Logf("Creating namespace %s", newNamespace.ObjectMeta.Name)
			err = r.Create(ctx, newNamespace)
			if err != nil {
				t.Fatal(err)
			}

			// active wait for the configmap to be updated
			found := false
			for range 12 {
				err = r.Get(ctx, "instana-agent-namespaces", cfg.Namespace(), cm)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(cm.Data[yamlName], newNamespace.ObjectMeta.Name) {
					t.Logf("The namespace %s was present in the configmap", newNamespace.ObjectMeta.Name)
					if strings.Contains(cm.Data[yamlName], newNamespace.ObjectMeta.Labels["agentOperatorTest"]) {
						log.Infof("The expected label agentOperatorTest with value %s was found", newNamespace.ObjectMeta.Labels["agentOperatorTest"])
					} else {
						t.Errorf("Expected to find label agentOperatorTest with value %s", newNamespace.ObjectMeta.Labels["agentOperatorTest"])
					}
					found = true
					break
				} else {
					t.Log("Give the operator a few more seconds to inject resources")
					time.Sleep(5 * time.Second)
				}
			}
			if !found {
				t.Error("Could not find the new namespace in the configmap")
				t.Error("=== Dumping configMap content as an error occured ===")
				t.Error(cm.Data[yamlName])
				t.Error("=== Dumping operator logs as an error occured ===")
				p := utils.RunCommand(
					"kubectl logs deployment/instana-agent-controller-manager",
				)
				t.Error(
					"Operator logs", p.Command(), p.Err(), p.Out(), p.ExitCode(),
				)
			}

			newNamespace.ObjectMeta.Labels["agentOperatorTest2"] = envconf.RandomName("operator-e2e-label", 30)
			// Update the new namespace with new label
			t.Logf("Update namespace label for %s", newNamespace.ObjectMeta.Name)
			err = r.Get(ctx, newNamespace.Name, cfg.Namespace(), newNamespace)
			if err != nil {
				t.Fatal(err)
			}

			err = r.Update(ctx, newNamespace)
			if err != nil {
				t.Fatal(err)
			}

			// active wait for the configmap to be updated
			found = false
			for range 12 {
				err = r.Get(ctx, "instana-agent-namespaces", cfg.Namespace(), cm)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(cm.Data[yamlName], newNamespace.ObjectMeta.Name) {
					t.Logf("The namespace %s was present in the configmap", newNamespace.ObjectMeta.Name)
					if !strings.Contains(cm.Data[yamlName], newNamespace.ObjectMeta.Labels["agentOperatorTest2"]) {
						t.Errorf("The expected to find label agentOperatorTest2 with value %s", newNamespace.ObjectMeta.Labels["agentOperatorTest2"])
					}
					found = true
					break
				} else {
					t.Log("Give the operator a few more seconds to inject resources")
					time.Sleep(5 * time.Second)
				}
			}
			if !found {
				t.Errorf("Could not find the updated label agentOperatorTest2 with value %s in the configmap", newNamespace.ObjectMeta.Labels["agentOperatorTest2"])
				t.Error("=== Dumping configMap content as an error occured ===")
				t.Error(cm.Data[yamlName])
				t.Error("=== Dumping operator logs as an error occured ===")
				p := utils.RunCommand(
					"kubectl logs deployment/instana-agent-controller-manager",
				)
				t.Error(
					"Operator logs", p.Command(), p.Err(), p.Out(), p.ExitCode(),
				)
			}

			// Cleanup the new namespace with random name again
			t.Logf("Deleting namespace %s", newNamespace.ObjectMeta.Name)
			err = r.Delete(ctx, newNamespace)
			if err != nil {
				t.Fatal(err)
			}

			found = true
			for range 12 {
				err = r.Get(ctx, "instana-agent-namespaces", cfg.Namespace(), cm)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(cm.Data[yamlName], newNamespace.ObjectMeta.Name) {
					t.Logf("The namespace %s was no longer present in the configmap", newNamespace.ObjectMeta.Name)
					found = false
					break
				} else {
					t.Log("Give the operator a few more seconds to inject resources")
					time.Sleep(5 * time.Second)
				}
			}
			if found {
				t.Error("Could still find the new namespace in the configmap, even if it was deleted")
				t.Error("=== Dumping configMap content as an error occured ===")
				t.Error(cm.Data[yamlName])
				t.Error("=== Dumping operator logs as an error occured ===")
				p := utils.RunCommand(
					"kubectl logs deployment/instana-agent-controller-manager",
				)
				t.Error(
					"Operator logs", p.Command(), p.Err(), p.Out(), p.ExitCode(),
				)
			}

			return ctx
		}).
		Feature()

	// test feature
	testEnv.Test(t, installAndCheckNamespaceLabels)
}
