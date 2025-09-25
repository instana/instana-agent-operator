/*
(c) Copyright IBM Corp. 2025

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

package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/instana/instana-agent-operator/pkg/pointer"
)

func TestInstanaAgent_Default_ETCDSettings(t *testing.T) {
	// Given
	agent := &InstanaAgent{}

	// When
	agent.Default()

	// Then
	assert.NotNil(t, agent.Spec.K8sSensor.ETCD.Insecure, "ETCD.Insecure should be defaulted")
	assert.False(t, *agent.Spec.K8sSensor.ETCD.Insecure, "ETCD.Insecure should default to false")
	assert.Equal(
		t,
		"/var/run/secrets/kubernetes.io/serviceaccount",
		agent.Spec.K8sSensor.ETCD.CA.MountPath,
		"ETCD.CA.MountPath should default to service account path",
	)
}

func TestInstanaAgent_Default_ETCDSettings_PreserveUserValues(t *testing.T) {
	// Given
	agent := &InstanaAgent{
		Spec: InstanaAgentSpec{
			K8sSensor: K8sSpec{
				ETCD: ETCDSpec{
					Insecure: pointer.To(true),
					CA: CASpec{
						MountPath: "/custom/path",
					},
				},
			},
		},
	}

	// When
	agent.Default()

	// Then
	assert.True(
		t,
		*agent.Spec.K8sSensor.ETCD.Insecure,
		"User-provided ETCD.Insecure value should be preserved",
	)
	assert.Equal(
		t,
		"/custom/path",
		agent.Spec.K8sSensor.ETCD.CA.MountPath,
		"User-provided ETCD.CA.MountPath should be preserved",
	)
}
