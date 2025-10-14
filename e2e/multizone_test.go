/*
 * (c) Copyright IBM Corp. 2024
 * (c) Copyright Instana Inc. 2024
 */

package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/pkg/utils"
)

// VerifyDaemonSetsExistButNotScheduled checks that the specified DaemonSets exist but have no pods scheduled
// and verifies they have the correct node selectors
func VerifyDaemonSetsExistButNotScheduled(daemonSetNames ...string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}

		// Wait briefly to ensure DaemonSets are fully created and stabilized
		time.Sleep(5 * time.Second)

		for _, dsName := range daemonSetNames {
			var ds appsv1.DaemonSet
			err := r.Get(ctx, dsName, InstanaNamespace, &ds)

			if err != nil {
				t.Fatalf("Failed to get DaemonSet %s: %v", dsName, err)
			}

			// Only log detailed information if there's an issue
			if ds.Status.CurrentNumberScheduled > 0 {
				t.Logf(
					"DaemonSet %s has %d scheduled pods when it should have 0",
					dsName,
					ds.Status.CurrentNumberScheduled,
				)
			}

			// Verify no pods are scheduled
			if ds.Status.CurrentNumberScheduled > 0 {
				// Debug: Print which nodes have the pods
				p := utils.RunCommand(
					fmt.Sprintf(
						"kubectl get pods -n %s -l app.kubernetes.io/name=instana-agent,instana.io/zone=%s -o wide",
						InstanaNamespace,
						strings.TrimPrefix(dsName, "instana-agent-"),
					),
				)
				t.Logf("Pods for DaemonSet %s: %s", dsName, p.Out())

				t.Fatalf("Expected DaemonSet %s to have 0 scheduled pods, but got %d",
					dsName, ds.Status.CurrentNumberScheduled)
			}

			// Verify node selector exists
			if ds.Spec.Template.Spec.Affinity == nil ||
				ds.Spec.Template.Spec.Affinity.NodeAffinity == nil ||
				ds.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
				t.Fatalf(
					"DaemonSet %s does not have the required node affinity configuration",
					dsName,
				)
			}

			// Check for the pool label in the node selector
			found := false
			nodeAffinity := ds.Spec.Template.Spec.Affinity.NodeAffinity
			requiredScheduling := nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution
			for _, term := range requiredScheduling.NodeSelectorTerms {
				for _, expr := range term.MatchExpressions {
					if expr.Key == "pool" {
						found = true
						break
					}
				}
				if found {
					break
				}
			}

			if !found {
				t.Fatalf("DaemonSet %s does not have the expected pool node selector", dsName)
			}
		}

		return ctx
	}
}

// VerifyDaemonSetScheduled checks that the specified DaemonSet has exactly one pod scheduled
func VerifyDaemonSetScheduled(daemonSetName string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}

		var ds appsv1.DaemonSet
		err = r.Get(ctx, daemonSetName, InstanaNamespace, &ds)

		if err != nil {
			t.Fatalf("Failed to get DaemonSet %s: %v", daemonSetName, err)
		}

		// Only log if there's an issue
		if ds.Status.CurrentNumberScheduled != 1 {
			t.Logf(
				"DaemonSet %s has %d scheduled pods, expected 1",
				daemonSetName,
				ds.Status.CurrentNumberScheduled,
			)
		}

		// Verify exactly one pod is scheduled
		if ds.Status.CurrentNumberScheduled != 1 {
			// Debug: Print node labels
			p := utils.RunCommand("kubectl get nodes --show-labels")
			t.Logf("Node labels: %s", p.Out())

			// Debug: Print DaemonSet details
			p = utils.RunCommand(
				fmt.Sprintf("kubectl describe ds %s -n %s", daemonSetName, InstanaNamespace),
			)
			t.Logf("DaemonSet details: %s", p.Out())

			t.Fatalf("Expected DaemonSet %s to have exactly 1 scheduled pod, but got %d",
				daemonSetName, ds.Status.CurrentNumberScheduled)
		}

		return ctx
	}
}

// WaitForAgentDaemonSetPodsScheduled waits for a DaemonSet to have exactly one pod scheduled
func WaitForAgentDaemonSetPodsScheduled(daemonSetName string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}

		// First, check if the DaemonSet exists and has the correct configuration
		var ds appsv1.DaemonSet
		err = r.Get(ctx, daemonSetName, InstanaNamespace, &ds)
		if err != nil {
			t.Fatalf("Failed to get DaemonSet %s: %v", daemonSetName, err)
		}

		// No need to log the node selector in normal operation

		// Check if the DaemonSet has node affinity
		if ds.Spec.Template.Spec.Affinity == nil ||
			ds.Spec.Template.Spec.Affinity.NodeAffinity == nil ||
			ds.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
			t.Fatalf(
				"DaemonSet %s does not have the required node affinity configuration",
				daemonSetName,
			)
		}

		// Poll until the DaemonSet has exactly one scheduled pod
		err = wait.For(func(ctx context.Context) (bool, error) {
			var updatedDs appsv1.DaemonSet
			err := r.Get(ctx, daemonSetName, InstanaNamespace, &updatedDs)

			if err != nil {
				return false, err
			}

			// Only log every 30 seconds to reduce verbosity
			if time.Now().Unix()%30 == 0 {
				t.Logf(
					"Waiting for DaemonSet %s pods (current: %d, ready: %d)",
					daemonSetName,
					updatedDs.Status.CurrentNumberScheduled,
					updatedDs.Status.NumberReady,
				)
			}

			// Wait until exactly one pod is scheduled
			return updatedDs.Status.CurrentNumberScheduled == 1, nil
		}, wait.WithTimeout(2*time.Minute), wait.WithInterval(5*time.Second))

		if err != nil {
			// Debug: Print node labels
			p := utils.RunCommand("kubectl get nodes --show-labels")
			t.Logf("Node labels: %s", p.Out())

			// Debug: Print DaemonSet details
			p = utils.RunCommand(
				fmt.Sprintf("kubectl describe ds %s -n %s", daemonSetName, InstanaNamespace),
			)
			t.Logf("DaemonSet details: %s", p.Out())

			// Check for any scheduling issues
			p = utils.RunCommand(
				fmt.Sprintf(
					"kubectl get events --field-selector involvedObject.name=%s -n %s",
					daemonSetName,
					InstanaNamespace,
				),
			)
			t.Logf("Events for DaemonSet %s: %s", daemonSetName, p.Out())

			t.Fatalf(
				"Failed waiting for DaemonSet %s to have exactly one scheduled pod: %v",
				daemonSetName,
				err,
			)
		}

		return ctx
	}
}

// WaitForAgentDaemonSetPodTermination waits for all pods of a DaemonSet to terminate
func WaitForAgentDaemonSetPodTermination(daemonSetName string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}

		// Poll until the DaemonSet has 0 scheduled pods
		err = wait.For(func(ctx context.Context) (bool, error) {
			var ds appsv1.DaemonSet
			err := r.Get(ctx, daemonSetName, InstanaNamespace, &ds)

			if err != nil {
				if errors.IsNotFound(err) {
					// DaemonSet not found, consider it terminated
					return true, nil
				}
				return false, err
			}

			// Only log every 30 seconds to reduce verbosity
			if time.Now().Unix()%30 == 0 {
				t.Logf(
					"Waiting for DaemonSet %s pods to terminate (current: %d)",
					daemonSetName,
					ds.Status.CurrentNumberScheduled,
				)
			}
			return ds.Status.CurrentNumberScheduled == 0, nil
		}, wait.WithTimeout(2*time.Minute), wait.WithInterval(5*time.Second))

		if err != nil {
			t.Fatalf("Failed waiting for DaemonSet %s pods to terminate: %v", daemonSetName, err)
		}

		return ctx
	}
}

// CleanupNodeLabels removes any pool labels from all nodes
func CleanupNodeLabels() features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		// Get all nodes
		nodeList := utils.FetchCommandOutput("kubectl get nodes -o name")
		nodes := strings.Split(strings.TrimSpace(nodeList), "\n")

		// Count how many nodes had labels removed for summary logging
		labelsRemoved := 0

		for _, nodeName := range nodes {
			node := strings.TrimPrefix(nodeName, "node/")

			// First check if the node has a pool label
			nodeLabels := utils.FetchCommandOutput(
				fmt.Sprintf("kubectl get node %s --show-labels", node),
			)

			// Only try to remove the label if it exists
			if strings.Contains(nodeLabels, "pool=") {
				p := utils.RunCommand(fmt.Sprintf("kubectl label node %s pool- --overwrite", node))
				if p.Err() != nil {
					t.Logf("Warning: Error removing pool label from node %s: %v", node, p.Err())
				} else {
					labelsRemoved++
				}
			}
		}

		// Verify all labels were removed
		allNodeLabels := utils.FetchCommandOutput("kubectl get nodes --show-labels")
		if strings.Contains(allNodeLabels, "pool=") {
			t.Logf("Warning: Some nodes still have pool labels after cleanup")
		} else if labelsRemoved > 0 {
			t.Logf("Removed pool labels from %d nodes", labelsRemoved)
		}

		return ctx
	}
}

func TestMultiZones(t *testing.T) {
	installWithMultiZones := features.New("multizone agent").
		Setup(SetupOperatorDevBuild()).
		Setup(WaitForDeploymentToBecomeReady(InstanaOperatorDeploymentName)).
		// Clean up any existing pool labels from all nodes
		Setup(CleanupNodeLabels()).
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			// Verify all nodes have no pool labels
			nodeLabels := utils.FetchCommandOutput("kubectl get nodes --show-labels")
			if strings.Contains(nodeLabels, "pool=") {
				t.Fatal("Found nodes with pool labels before test start - cleanup failed")
			}

			// 2. Create the multizone CR
			t.Logf("Creating agent CR with multizone enabled")
			err = decoder.ApplyWithManifestDir(
				ctx,
				r,
				"../config/samples",
				"instana_v1_multizone_instanaagent.yaml",
				[]resources.CreateOption{},
			)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("CR created")

			return ctx
		}).
		// Wait for k8sensor deployment to become ready
		Assess("wait for k8sensor deployment to become ready", WaitForDeploymentToBecomeReady(K8sensorDeploymentName)).

		// 3. Verify both DaemonSets are created but not scheduled
		Assess("verify daemonsets are created but not scheduled", VerifyDaemonSetsExistButNotScheduled(
			AgentDaemonSetName+"-e2e-test-pool-01",
			AgentDaemonSetName+"-e2e-test-pool-02",
		)).

		// 4. Test first zone
		Assess("test first zone", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// Get a worker node name (not a master/control-plane node)
			nodeList := utils.FetchCommandOutput(
				"kubectl get nodes -l '!node-role.kubernetes.io/master,!node-role.kubernetes.io/control-plane' -o name",
			)
			nodes := strings.Split(strings.TrimSpace(nodeList), "\n")
			if len(nodes) == 0 {
				t.Log("No dedicated worker nodes found, falling back to any node")
				nodeList = utils.FetchCommandOutput("kubectl get nodes -o name")
				nodes = strings.Split(strings.TrimSpace(nodeList), "\n")
			}
			node := strings.TrimPrefix(nodes[0], "node/")

			// Apply first label
			t.Log("Applying pool-01 label to node")
			cmd := utils.RunCommand(
				fmt.Sprintf("kubectl label node %s pool=pool-01 --overwrite", node),
			)
			if cmd.Err() != nil {
				t.Fatal("Error labeling node", cmd.Err(), cmd.Out(), cmd.ExitCode())
			}

			// No need to verify and log the labels in normal operation

			// Wait for first DaemonSet to have pods scheduled
			WaitForAgentDaemonSetPodsScheduled(AgentDaemonSetName+"-e2e-test-pool-01")(ctx, t, cfg)

			// Verify that exactly one pod is scheduled
			VerifyDaemonSetScheduled(AgentDaemonSetName+"-e2e-test-pool-01")(ctx, t, cfg)

			return ctx
		}).

		// 5. Ensure clean transition
		Assess("transition between zones", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// Get a worker node name (not a master/control-plane node)
			nodeList := utils.FetchCommandOutput(
				"kubectl get nodes -l '!node-role.kubernetes.io/master,!node-role.kubernetes.io/control-plane' -o name",
			)
			nodes := strings.Split(strings.TrimSpace(nodeList), "\n")
			if len(nodes) == 0 {
				t.Log("No dedicated worker nodes found, falling back to any node")
				nodeList = utils.FetchCommandOutput("kubectl get nodes -o name")
				nodes = strings.Split(strings.TrimSpace(nodeList), "\n")
			}
			node := strings.TrimPrefix(nodes[0], "node/")

			// Remove first label
			t.Log("Removing pool label from node")
			cmd := utils.RunCommand(fmt.Sprintf("kubectl label node %s pool-", node))
			if cmd.Err() != nil {
				t.Fatal("Error removing label from node", cmd.Err(), cmd.Out(), cmd.ExitCode())
			}

			// No need to verify and log the labels in normal operation

			// Wait for first DaemonSet pod to terminate completely
			t.Log("Waiting for first zone pod to terminate")
			WaitForAgentDaemonSetPodTermination(AgentDaemonSetName+"-e2e-test-pool-01")(ctx, t, cfg)
			return ctx
		}).

		// 6. Test second zone
		Assess("test second zone", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// Get a worker node name (not a master/control-plane node)
			nodeList := utils.FetchCommandOutput(
				"kubectl get nodes -l '!node-role.kubernetes.io/master,!node-role.kubernetes.io/control-plane' -o name",
			)
			nodes := strings.Split(strings.TrimSpace(nodeList), "\n")
			if len(nodes) == 0 {
				t.Log("No dedicated worker nodes found, falling back to any node")
				nodeList = utils.FetchCommandOutput("kubectl get nodes -o name")
				nodes = strings.Split(strings.TrimSpace(nodeList), "\n")
			}
			node := strings.TrimPrefix(nodes[0], "node/")

			// Apply second label
			t.Log("Applying pool-02 label to node")
			cmd := utils.RunCommand(
				fmt.Sprintf("kubectl label node %s pool=pool-02 --overwrite", node),
			)
			if cmd.Err() != nil {
				t.Fatal("Error labeling node", cmd.Err(), cmd.Out(), cmd.ExitCode())
			}

			// No need to verify and log the labels in normal operation

			// Wait for second DaemonSet to have pods scheduled
			WaitForAgentDaemonSetPodsScheduled(AgentDaemonSetName+"-e2e-test-pool-02")(ctx, t, cfg)

			// Verify that exactly one pod is scheduled
			VerifyDaemonSetScheduled(AgentDaemonSetName+"-e2e-test-pool-02")(ctx, t, cfg)

			return ctx
		}).
		Teardown(CleanupNodeLabels()).
		Feature()

	// test feature
	testEnv.Test(t, installWithMultiZones)
}

// Made with Bob
