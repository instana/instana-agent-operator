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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	backend "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/backends"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/status"
)

func countEnvVars(envVars []corev1.EnvVar, name string) int {
	count := 0
	for _, env := range envVars {
		if env.Name == name {
			count++
		}
	}
	return count
}

func countVolumes(volumes []corev1.Volume, name string) int {
	count := 0
	for _, vol := range volumes {
		if vol.Name == name {
			count++
		}
	}
	return count
}

func countVolumeMounts(mounts []corev1.VolumeMount, name string) int {
	count := 0
	for _, mount := range mounts {
		if mount.Name == name {
			count++
		}
	}
	return count
}

// agentWithCustomETCDCA returns an agent that configures its own ETCD CA, without
// pinning the targets, so ETCD discovery still runs and reports a CA of its own.
func agentWithCustomETCDCA() *instanav1.InstanaAgent {
	return &instanav1.InstanaAgent{
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
			K8sSensor: instanav1.K8sSpec{
				ETCD: instanav1.ETCDSpec{
					CA: instanav1.CASpec{
						SecretName: "custom-etcd-ca",
						MountPath:  "/custom/etcd",
					},
				},
			},
		},
	}
}

// TestDeploymentBuilder_CustomETCDCAWinsOverDiscoveredCA covers a CR that configures
// its own ETCD CA on a cluster where discovery also finds one. Both used to be
// emitted, giving the container two ETCD_CA_FILE entries and the pod two volumes
// named etcd-ca, which the API server rejects as duplicates.
func TestDeploymentBuilder_CustomETCDCAWinsOverDiscoveredCA(t *testing.T) {
	// Given
	agent := agentWithCustomETCDCA()

	mockStatusManager := &status.MockAgentStatusManager{}
	backendObj := backend.NewK8SensorBackend("", "test-key", "", "test-host", "443")

	deploymentContext := &DeploymentContext{
		DiscoveredETCDTargets: []string{"https://10.0.0.1:2379/metrics"},
		ETCDCASecretName:      constants.ETCDCASecretName,
	}

	builder := NewDeploymentBuilder(
		agent,
		false,
		mockStatusManager,
		*backendObj,
		nil,
		deploymentContext,
	).(*deploymentBuilder)

	// When
	envVars := builder.getEnvVars()
	volumes, mounts := builder.getVolumes()

	// Then - the CR configuration wins and nothing is emitted twice
	assert.Equal(
		t,
		1,
		countEnvVars(envVars, constants.EnvETCDCAFile),
		"ETCD_CA_FILE should be set exactly once",
	)
	assert.Equal(
		t,
		"/custom/etcd/ca.crt",
		findEnvVar(envVars, constants.EnvETCDCAFile).Value,
		"the CA file configured on the CR should win over the discovered one",
	)

	assert.Equal(t, 1, countVolumes(volumes, "etcd-ca"), "there should be one etcd-ca volume")
	assert.Equal(
		t,
		"custom-etcd-ca",
		findVolume(volumes, "etcd-ca").Secret.SecretName,
		"the CA secret configured on the CR should win over the discovered one",
	)
	assert.Equal(
		t,
		1,
		countVolumeMounts(mounts, "etcd-ca"),
		"there should be one etcd-ca mount",
	)

	// The discovered targets are still applied, only the CA is left to the CR
	assert.Equal(
		t,
		"https://10.0.0.1:2379/metrics",
		findEnvVar(envVars, constants.EnvETCDTargets).Value,
		"discovered targets should still be applied",
	)
}

// TestDeploymentBuilder_DiscoveredETCDCAUsedWithoutCustomCA is the counterpart: with
// no CA on the CR, the discovered one is still mounted as before.
func TestDeploymentBuilder_DiscoveredETCDCAUsedWithoutCustomCA(t *testing.T) {
	// Given
	agent := agentWithCustomETCDCA()
	agent.Spec.K8sSensor.ETCD.CA = instanav1.CASpec{}

	mockStatusManager := &status.MockAgentStatusManager{}
	backendObj := backend.NewK8SensorBackend("", "test-key", "", "test-host", "443")

	deploymentContext := &DeploymentContext{
		DiscoveredETCDTargets: []string{"https://10.0.0.1:2379/metrics"},
		ETCDCASecretName:      constants.ETCDCASecretName,
	}

	builder := NewDeploymentBuilder(
		agent,
		false,
		mockStatusManager,
		*backendObj,
		nil,
		deploymentContext,
	).(*deploymentBuilder)

	// When
	envVars := builder.getEnvVars()
	volumes, mounts := builder.getVolumes()

	// Then
	assert.Equal(
		t,
		1,
		countEnvVars(envVars, constants.EnvETCDCAFile),
		"ETCD_CA_FILE should be set exactly once",
	)
	assert.Equal(
		t,
		constants.ETCDCAMountPath+"/ca.crt",
		findEnvVar(envVars, constants.EnvETCDCAFile).Value,
		"the discovered CA file should be used when the CR configures none",
	)

	assert.Equal(t, 1, countVolumes(volumes, "etcd-ca"), "there should be one etcd-ca volume")
	assert.Equal(
		t,
		constants.ETCDCASecretName,
		findVolume(volumes, "etcd-ca").Secret.SecretName,
		"the discovered CA secret should be mounted",
	)
	assert.Equal(
		t,
		1,
		countVolumeMounts(mounts, "etcd-ca"),
		"there should be one etcd-ca mount",
	)
}
