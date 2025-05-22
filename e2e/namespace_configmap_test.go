/*
 * (c) Copyright IBM Corp. 2024
 * (c) Copyright Instana Inc. 2024
 */

package e2e

import (
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestNamespaceLabelConfigmap(t *testing.T) {
	installAndCheckNamespaceLabels := features.New("check namespace configmap in agent pods").
		Setup(SetupOperatorDevBuild()).
		Assess("wait for instana-agent-controller-manager deployment to become ready", WaitForDeploymentToBecomeReady(InstanaOperatorDeploymentName)).
		Assess("wait for k8sensor deployment to become ready", WaitForDeploymentToBecomeReady(K8sensorDeploymentName)).
		Assess("wait for agent daemonset to become ready", WaitForAgentDaemonSetToBecomeReady()).
		Assess("check agent log for successful connection", WaitForAgentSuccessfulBackendConnection()).
		Assess("check agent log for successful connection", ValidateAgentNamespacesLabelConfigmapConfiguration(
			`instana-agent:
        labels:
            kubernetes.io/metadata.name: instana-agent`)).
		Feature()

	// test feature
	testEnv.Test(t, installAndCheckNamespaceLabels)
}
