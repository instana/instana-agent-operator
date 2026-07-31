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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	backend "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/backends"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/status"
)

func agentForETCDEnvTests() *instanav1.InstanaAgent {
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
		},
	}
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
