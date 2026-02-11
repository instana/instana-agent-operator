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

package daemonset

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/status"
)

func TestDaemonSetBuilder_AppendFQDNToAgentID(t *testing.T) {
	tests := []struct {
		name                      string
		shouldSetAppendFQDNEnvVar bool
		expectedEnvVarPresent     bool
		expectedEnvVarValue       string
	}{
		{
			name:                      "Should set INSTANA_APPEND_FQDN_TO_AGENT_ID when flag is true",
			shouldSetAppendFQDNEnvVar: true,
			expectedEnvVarPresent:     true,
			expectedEnvVarValue:       "true",
		},
		{
			name:                      "Should not set INSTANA_APPEND_FQDN_TO_AGENT_ID when flag is false",
			shouldSetAppendFQDNEnvVar: false,
			expectedEnvVarPresent:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			agent := &instanav1.InstanaAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "test-namespace",
				},
				Spec: instanav1.InstanaAgentSpec{
					Agent: instanav1.BaseAgentSpec{
						Key:          "test-key",
						EndpointHost: "test-host",
						EndpointPort: "test-port",
					},
					Cluster: instanav1.Name{
						Name: "test-cluster",
					},
				},
			}

			statusManager := status.NewAgentStatusManager(nil, nil)
			dsBuilder := NewDaemonSetBuilder(
				agent,
				false,
				statusManager,
				tt.shouldSetAppendFQDNEnvVar,
			)

			// When
			obj := dsBuilder.Build()
			require.True(t, obj.IsPresent(), "Expected DaemonSet to be built")

			// Then
			ds, ok := obj.Get().(*appsv1.DaemonSet)
			require.True(t, ok, "Expected DaemonSet object")

			envVars := ds.Spec.Template.Spec.Containers[0].Env

			// Check if INSTANA_APPEND_FQDN_TO_AGENT_ID is present
			found := false
			var actualValue string
			for _, env := range envVars {
				if env.Name == "INSTANA_APPEND_FQDN_TO_AGENT_ID" {
					found = true
					actualValue = env.Value
					break
				}
			}

			assert.Equal(t, tt.expectedEnvVarPresent, found,
				"INSTANA_APPEND_FQDN_TO_AGENT_ID presence mismatch")

			if tt.expectedEnvVarPresent {
				assert.Equal(t, tt.expectedEnvVarValue, actualValue,
					"INSTANA_APPEND_FQDN_TO_AGENT_ID value mismatch")
			}
		})
	}
}

func TestDaemonSetBuilder_AppendFQDNToAgentID_WithPodEnv(t *testing.T) {
	// Given - agent with pod.env that should take precedence
	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
		},
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				Key:          "test-key",
				EndpointHost: "test-host",
				EndpointPort: "test-port",
				Pod: instanav1.AgentPodSpec{
					Env: []corev1.EnvVar{
						{
							Name:  "INSTANA_APPEND_FQDN_TO_AGENT_ID",
							Value: "false",
						},
					},
				},
			},
			Cluster: instanav1.Name{
				Name: "test-cluster",
			},
		},
	}

	statusManager := status.NewAgentStatusManager(nil, nil)
	// Even though we set the flag to true, pod.env should take precedence
	dsBuilder := NewDaemonSetBuilder(agent, false, statusManager, true)

	// When
	obj := dsBuilder.Build()
	require.True(t, obj.IsPresent(), "Expected DaemonSet to be built")

	// Then
	ds, ok := obj.Get().(*appsv1.DaemonSet)
	require.True(t, ok, "Expected DaemonSet object")

	envVars := ds.Spec.Template.Spec.Containers[0].Env

	// Check that pod.env value takes precedence
	found := false
	var actualValue string
	for _, env := range envVars {
		if env.Name == "INSTANA_APPEND_FQDN_TO_AGENT_ID" {
			found = true
			actualValue = env.Value
			break
		}
	}

	assert.True(t, found, "INSTANA_APPEND_FQDN_TO_AGENT_ID should be present")
	assert.Equal(t, "false", actualValue,
		"pod.env value should take precedence over builder flag")
}

// Made with Bob
