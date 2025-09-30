//go:build multinode
// +build multinode

/*
 * (c) Copyright IBM Corp. 2024
 * (c) Copyright Instana Inc. 2024
 */

package e2e

import (
	"context"
	"fmt"
	"log"
	"strings"
	"testing"

	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/pkg/utils"
)

func TestMultiZones(t *testing.T) {
	installWithMultiZones := features.New("multizone agent").
		Setup(SetupOperatorDevBuild()).
		Setup(WaitForDeploymentToBecomeReady(InstanaOperatorDeploymentName)).
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			t.Log("Labeling the nodes")
			nodeList := utils.FetchCommandOutput("kubectl get nodes -o name")
			nodes := strings.Split(strings.TrimSpace(nodeList), "\n")
			if len(nodes) < 2 {
				log.Fatalf("Not enough nodes found for testing.")
			}
			node1 := strings.TrimPrefix(nodes[0], "node/")
			node2 := strings.TrimPrefix(nodes[1], "node/")

			p := utils.RunCommand(
				fmt.Sprintf(
					`kubectl label node %s %s`,
					node1,
					"pool=pool-01",
				),
			)
			if p.Err() != nil {
				t.Fatal("Error during labeling the nodes", p.Err(), p.Out(), p.ExitCode())
			}

			p = utils.RunCommand(
				fmt.Sprintf(
					`kubectl label nodes %s %s`,
					node2,
					"pool=pool-02",
				),
			)
			if p.Err() != nil {
				t.Fatal("Error during labeling the nodes", p.Err(), p.Out(), p.ExitCode())
			}

			t.Logf("Creating dummy agent CR with multizone enabled")
			err = decoder.ApplyWithManifestDir(ctx, r, "../config/samples", "instana_v1_multizone_instanaagent.yaml", []resources.CreateOption{})
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("CR created")

			return ctx
		}).
		Assess("wait for first k8sensor deployment to become ready", WaitForDeploymentToBecomeReady(K8sensorDeploymentName)).
		Assess("wait for agent daemonset in the first zone to become ready", WaitForAgentDaemonSetToBecomeReady(AgentDaemonSetName+"-e2e-test-pool-01")).
		Assess("wait for agent daemonset in the second zone to become ready", WaitForAgentDaemonSetToBecomeReady(AgentDaemonSetName+"-e2e-test-pool-02")).
		Feature()

	// test feature
	testEnv.Test(t, installWithMultiZones)
}
