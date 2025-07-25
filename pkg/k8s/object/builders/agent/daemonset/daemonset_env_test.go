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

package daemonset

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/status"
)

func TestDaemonSetBuilder_PodEnvVars(t *testing.T) {
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
				Pod: instanav1.AgentPodSpec{
					Env: []corev1.EnvVar{
						{
							Name:  "TEST_ENV_VAR",
							Value: "test-value",
						},
						{
							Name: "TEST_ENV_VAR_FROM_FIELD",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "metadata.name",
								},
							},
						},
						{
							Name: "TEST_ENV_VAR_FROM_SECRET",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "test-secret",
									},
									Key: "test-key",
								},
							},
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
	dsBuilder := NewDaemonSetBuilder(agent, false, statusManager)

	// When
	obj := dsBuilder.Build()
	if !obj.IsPresent() {
		t.Fatal("Expected DaemonSet to be built")
	}

	// Then
	ds, ok := obj.Get().(*appsv1.DaemonSet)
	if !ok {
		t.Fatal("Expected DaemonSet object")
	}

	envVars := ds.Spec.Template.Spec.Containers[0].Env

	// Check if our custom environment variables are present
	foundTestEnvVar := false
	foundTestEnvVarFromField := false
	foundTestEnvVarFromSecret := false

	for _, env := range envVars {
		if env.Name == "TEST_ENV_VAR" {
			foundTestEnvVar = true
			assert.Equal(t, "test-value", env.Value)
		}
		if env.Name == "TEST_ENV_VAR_FROM_FIELD" {
			foundTestEnvVarFromField = true
			assert.Equal(t, "metadata.name", env.ValueFrom.FieldRef.FieldPath)
		}
		if env.Name == "TEST_ENV_VAR_FROM_SECRET" {
			foundTestEnvVarFromSecret = true
			assert.Equal(t, "test-secret", env.ValueFrom.SecretKeyRef.LocalObjectReference.Name)
			assert.Equal(t, "test-key", env.ValueFrom.SecretKeyRef.Key)
		}
	}

	assert.True(t, foundTestEnvVar, "TEST_ENV_VAR not found in container environment variables")
	assert.True(t, foundTestEnvVarFromField, "TEST_ENV_VAR_FROM_FIELD not found in container environment variables")
	assert.True(t, foundTestEnvVarFromSecret, "TEST_ENV_VAR_FROM_SECRET not found in container environment variables")
}

func TestDaemonSetBuilder_EnvVarPrecedence(t *testing.T) {
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
				// Legacy env vars
				Env: map[string]string{
					"INSTANA_AGENT_TAGS": "legacy,simple,key-value",
					"DUPLICATE_ENV_VAR":  "legacy-value",
				},
				Pod: instanav1.AgentPodSpec{
					// New env vars with one duplicate to test precedence
					Env: []corev1.EnvVar{
						{
							Name:  "INSTANA_AGENT_TAGS",
							Value: "kubernetes,production,custom",
						},
						{
							Name:  "DUPLICATE_ENV_VAR",
							Value: "pod-value",
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
	dsBuilder := NewDaemonSetBuilder(agent, false, statusManager)

	// When
	obj := dsBuilder.Build()
	if !obj.IsPresent() {
		t.Fatal("Expected DaemonSet to be built")
	}

	// Then
	ds, ok := obj.Get().(*appsv1.DaemonSet)
	if !ok {
		t.Fatal("Expected DaemonSet object")
	}

	envVars := ds.Spec.Template.Spec.Containers[0].Env

	// Count occurrences of each env var name to check for duplicates
	envVarCounts := make(map[string]int)
	envVarValues := make(map[string]string)

	for _, env := range envVars {
		envVarCounts[env.Name]++
		if env.Value != "" {
			envVarValues[env.Name] = env.Value
		}
	}

	// Check that there are no duplicates
	for name, count := range envVarCounts {
		assert.Equal(t, 1, count, "Environment variable %s appears %d times, expected once", name, count)
	}

	// Check that pod.env values take precedence over agent.env values
	assert.Equal(t, "kubernetes,production,custom", envVarValues["INSTANA_AGENT_TAGS"],
		"pod.env value for INSTANA_AGENT_TAGS should take precedence")
	assert.Equal(t, "pod-value", envVarValues["DUPLICATE_ENV_VAR"],
		"pod.env value for DUPLICATE_ENV_VAR should take precedence")
}

// Made with Bob
