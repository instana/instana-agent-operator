/*
 * (c) Copyright IBM Corp. 2025
 */

package e2e

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/namespaces"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	log "k8s.io/klog/v2"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/support/utils"
)

func TestNamespaceLabelConfigmap(t *testing.T) {
	agent := NewAgentCr()
	installAndCheckNamespaceLabels := features.New("check namespace configmap in agent pods").
		Setup(SetupOperatorDevBuild()).
		Setup(DeployAgentCr(&agent)).
		Assess("wait for instana-agent-controller-manager deployment to become ready", WaitForDeploymentToBecomeReady(InstanaOperatorDeploymentName)).
		Assess("wait for k8sensor deployment to become ready", WaitForDeploymentToBecomeReady(K8sensorDeploymentName)).
		Assess("wait for agent daemonset to become ready", WaitForAgentDaemonSetToBecomeReady()).
		Assess("check agent pod for namespaces label file", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			log.Infof("Validating namespace file is available in agent daemonset with the given env var as path")
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
				[]string{"bash", "-c", "cat ${NAMESPACES_DETAILS_PATH} | grep -A 5 'namespaces:'"},
				&stdout,
				&stderr,
			); err != nil {
				t.Log(stderr.String())
				t.Error(err)
			}
			t.Logf("ExecInPod returned expected namespace file")
			return ctx
		}).
		Assess("check if configmap gets updated when label gets added, updated or removed from namespace", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			log.Infof("Validating unlabeled namespace is not found in configmap")
			// Create a client to interact with the Kube API
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}
			expectNamespaceNotPresentInConfigMap(ctx, r, t, cfg)

			log.Info("Adding label instana-workload-monitoring=true to instana-agent namespace, expect to find the namespace in configmap")
			p := utils.RunCommand(
				fmt.Sprintf("kubectl label ns %s instana-workload-monitoring=true", cfg.Namespace()),
			)
			if p.Err() != nil {
				t.Error(
					"Could not add label to namespace", p.Command(), p.Err(), p.Out(), p.ExitCode(),
				)
			}
			expectNamespaceWithLabelInConfigMap(ctx, r, t, cfg, "true")

			log.Info("Adding label instana-workload-monitoring=false to instana-agent namespace, expect to find the namespace in configmap")
			p = utils.RunCommand(
				fmt.Sprintf("kubectl label ns %s instana-workload-monitoring=false --overwrite", cfg.Namespace()),
			)
			if p.Err() != nil {
				t.Error(
					"Could not update label of namespace", p.Command(), p.Err(), p.Out(), p.ExitCode(),
				)
			}
			expectNamespaceWithLabelInConfigMap(ctx, r, t, cfg, "false")

			log.Info("Removing label instana-workload-monitoring entirely from instana-agent namespace, expect to not find the namespace in configmap")
			p = utils.RunCommand(
				fmt.Sprintf("kubectl label ns %s instana-workload-monitoring-", cfg.Namespace()),
			)
			if p.Err() != nil {
				t.Error(
					"Could not remove label from namespace", p.Command(), p.Err(), p.Out(), p.ExitCode(),
				)
			}

			expectNamespaceNotPresentInConfigMap(ctx, r, t, cfg)

			return ctx
		}).
		Feature()

	// test feature
	testEnv.Test(t, installAndCheckNamespaceLabels)
}

func expectNamespaceWithLabelInConfigMap(ctx context.Context, r *resources.Resources, t *testing.T, cfg *envconf.Config, expectedLabelValue string) {
	found := false
	for range 12 {
		value, _ := getLabelFromConfigmap(ctx, r, t, cfg)
		if value == expectedLabelValue {
			t.Logf("The namespace %s was present in the configmap and carried the label value %s", cfg.Namespace(), expectedLabelValue)
			found = true
			break
		} else {
			t.Log("Give the operator a few more seconds to update resources")
			time.Sleep(5 * time.Second)
		}
	}
	if !found {
		t.Error("Could still not find the namespace with the label in the configmap, even if it was labeled")
	}
}

func expectNamespaceNotPresentInConfigMap(ctx context.Context, r *resources.Resources, t *testing.T, cfg *envconf.Config) {
	found := true
	for range 12 {
		_, err := getLabelFromConfigmap(ctx, r, t, cfg)
		if err != nil {
			t.Logf("The namespace %s was removed from the configmap", cfg.Namespace())
			found = false
			break
		} else {
			t.Log("Give the operator a few more seconds to update resources")
			time.Sleep(5 * time.Second)
		}
	}
	if found {
		t.Error("Could still find the namespace in the configmap, even if the label was deleted")
	}
}

func getLabelFromConfigmap(ctx context.Context, r *resources.Resources, t *testing.T, cfg *envconf.Config) (string, error) {
	// check that new namespace with random values is not present in the configmap yet
	yamlName := "namespaces.yaml"
	cm := &corev1.ConfigMap{}
	err := r.Get(ctx, "instana-agent-namespaces", cfg.Namespace(), cm)
	if err != nil {
		t.Fatal(err)
	}
	var namespaceListFromConfigMap namespaces.NamespacesDetails
	if err := yaml.Unmarshal([]byte(cm.Data[yamlName]), &namespaceListFromConfigMap); err != nil {
		t.Fatal(err)
	}
	namespace, ok := namespaceListFromConfigMap.Namespaces[cfg.Namespace()]
	if !ok {
		return "", errors.New("missing namespace")
	}
	return namespace.Labels["instana-workload-monitoring"], nil
}
