/*
(c) Copyright IBM Corp. 2026

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package deployment

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	backend "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/backends"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/status"
)

// agentForETCDEnvTests returns an agent defaulted the same way the controller defaults
// it before the builders run. Skipping Default() here would test a CR shape that cannot
// reach the builder, and would hide anything that depends on a defaulted field, such as
// the ETCD CA mount path.
func agentForETCDEnvTests() *instanav1.InstanaAgent {
	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-ns",
		},
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				Key: "test-key",
			},
			Zone: instanav1.Name{
				Name: "test-zone",
			},
		},
	}
	agent.Default()
	return agent
}

func builderFor(
	agent *instanav1.InstanaAgent,
	isOpenShift bool,
	deploymentContext *DeploymentContext,
) *deploymentBuilder {
	backendObj := backend.NewK8SensorBackend("", "test-key", "", "test-host", "443")
	return NewDeploymentBuilder(
		agent,
		isOpenShift,
		&status.MockAgentStatusManager{},
		*backendObj,
		nil,
		deploymentContext,
	).(*deploymentBuilder)
}

// TestDeploymentBuilder_DiscoveryDoesNotSetETCDTargets covers that the operator no
// longer injects endpoints the k8sensor does not read. The k8sensor discovers the
// etcd endpoints itself, so ETCD_TARGETS was dead weight whose churn rolled the pod.
func TestDeploymentBuilder_DiscoveryDoesNotSetETCDTargets(t *testing.T) {
	// Given a cluster where discovery found a CA
	agent := agentForETCDEnvTests()
	builder := builderFor(agent, false, &DeploymentContext{
		ETCDCASecretName: constants.ETCDCASecretName,
	})

	// When
	envVars := builder.getEnvVars()

	// Then the CA is passed through but no endpoints are
	assert.Nil(
		t,
		findEnvVar(envVars, constants.EnvETCDTargets),
		"ETCD_TARGETS should not be set from discovery, the k8sensor finds the endpoints itself",
	)

	// ETCD_CA_FILE has to be set exactly once, and to the path the volume is mounted
	// at. The CR's CA mount path is defaulted to the service account path, so getting
	// this wrong points the k8sensor at a CA that does not sign the etcd serving cert
	// and every scrape fails TLS verification.
	caFile := findEnvVar(envVars, constants.EnvETCDCAFile)
	require.NotNil(t, caFile, "ETCD_CA_FILE should still be set, the k8sensor does read it")
	assert.Equal(t, constants.ETCDCAMountPath+"/ca.crt", caFile.Value)
	assert.Equal(
		t,
		1,
		countEnvVarsNamed(envVars, constants.EnvETCDCAFile),
		"ETCD_CA_FILE should be set exactly once",
	)

	// The env var must line up with where the volume actually puts the file
	_, mounts := builder.getVolumes()
	mount := findVolumeMount(mounts, "etcd-ca")
	require.NotNil(t, mount, "the etcd-ca volume should be mounted")
	assert.Equal(
		t,
		constants.ETCDCAMountPath,
		mount.MountPath,
		"ETCD_CA_FILE must point inside the etcd-ca mount",
	)
}

// countEnvVarsNamed returns how many times the named env var appears.
func countEnvVarsNamed(envVars []corev1.EnvVar, name string) int {
	count := 0
	for _, env := range envVars {
		if env.Name == name {
			count++
		}
	}
	return count
}

// TestDeploymentBuilder_CRETCDTargetsIgnored covers the deprecated CR field. The
// k8sensor never read ETCD_TARGETS from either source, so targets set on the CR are
// accepted for backwards compatibility but produce no env var.
func TestDeploymentBuilder_CRETCDTargetsIgnored(t *testing.T) {
	// Given
	agent := agentForETCDEnvTests()
	agent.Spec.K8sSensor.ETCD.Targets = []string{"https://etcd-1:2379"}
	builder := builderFor(agent, false, nil)

	// When
	envVars := builder.getEnvVars()

	// Then
	assert.Nil(
		t,
		findEnvVar(envVars, constants.EnvETCDTargets),
		"targets set on the CR are deprecated and should not produce ETCD_TARGETS",
	)
}

// TestDeploymentBuilder_CRETCDTargetsDoNotSuppressTheCA guards the interaction between
// the deprecated field and CA discovery. Targets on the CR used to short circuit
// discovery, so leaving that in place would have silently dropped the CA mount for
// anyone still setting them.
func TestDeploymentBuilder_CRETCDTargetsDoNotSuppressTheCA(t *testing.T) {
	// Given targets on the CR and a CA found by discovery
	agent := agentForETCDEnvTests()
	agent.Spec.K8sSensor.ETCD.Targets = []string{"https://etcd-1:2379"}
	builder := builderFor(agent, false, &DeploymentContext{
		ETCDCASecretName: constants.ETCDCASecretName,
	})

	// When
	envVars := builder.getEnvVars()
	volumes, mounts := builder.getVolumes()

	// Then the CA is still wired up
	assert.NotNil(
		t,
		findEnvVar(envVars, constants.EnvETCDCAFile),
		"the discovered CA should still be applied when the deprecated field is set",
	)
	assert.NotNil(t, findVolume(volumes, "etcd-ca"), "the etcd-ca volume should still be added")
	assert.NotNil(t, findVolumeMount(mounts, "etcd-ca"), "the etcd-ca mount should still be added")
}

// TestDeploymentBuilder_OpenShiftOmitsUnreadETCDEnvVars covers the OpenShift side.
// ETCD_METRICS_URL and ETCD_REQUEST_TIMEOUT are not read by the k8sensor either, so
// only the TLS settings are passed through.
func TestDeploymentBuilder_OpenShiftOmitsUnreadETCDEnvVars(t *testing.T) {
	// Given
	agent := agentForETCDEnvTests()
	builder := builderFor(agent, true, &DeploymentContext{
		OpenShiftETCDResourcesExist: true,
	})

	// When
	envVars := builder.getEnvVars()

	// Then
	assert.Nil(
		t,
		findEnvVar(envVars, constants.EnvETCDMetricsURL),
		"ETCD_METRICS_URL is not read by the k8sensor and should not be set",
	)
	assert.Nil(
		t,
		findEnvVar(envVars, constants.EnvETCDRequestTimeout),
		"ETCD_REQUEST_TIMEOUT is not read by the k8sensor and should not be set",
	)

	// The TLS settings are read, so they stay
	for _, name := range []string{
		constants.EnvETCDCAFile,
		constants.EnvETCDCertFile,
		constants.EnvETCDKeyFile,
	} {
		assert.NotNil(t, findEnvVar(envVars, name), "%s should still be set", name)
	}
}

// TestDeploymentBuilder_ETCDEnvIsStableAcrossBuilds is the regression guard for the
// reported restart loop: the same inputs must render the same env every time, so the
// server side apply is a no-op and the k8sensor pod is not rolled.
func TestDeploymentBuilder_ETCDEnvIsStableAcrossBuilds(t *testing.T) {
	// Given
	agent := agentForETCDEnvTests()
	deploymentContext := &DeploymentContext{ETCDCASecretName: constants.ETCDCASecretName}

	// When the same context is rendered on two consecutive reconciles
	firstEnv := builderFor(agent, false, deploymentContext).getEnvVars()
	firstVolumes, firstMounts := builderFor(agent, false, deploymentContext).getVolumes()

	secondEnv := builderFor(agent, false, deploymentContext).getEnvVars()
	secondVolumes, secondMounts := builderFor(agent, false, deploymentContext).getVolumes()

	// Then nothing moves
	assert.Equal(t, firstEnv, secondEnv, "env must be identical across reconciles")
	assert.Equal(t, firstVolumes, secondVolumes, "volumes must be identical across reconciles")
	assert.Equal(t, firstMounts, secondMounts, "mounts must be identical across reconciles")
}
