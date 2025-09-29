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

package controllers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
)

// TestGetSortedTargets tests the getSortedTargets helper function
func TestGetSortedTargets(t *testing.T) {
	testCases := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "Empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "Single item",
			input:    []string{"target1"},
			expected: []string{"target1"},
		},
		{
			name:     "Already sorted",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "Reverse order",
			input:    []string{"c", "b", "a"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "ETCD targets",
			input:    []string{"https://etcd-2:2379/metrics", "https://etcd-1:2379/metrics"},
			expected: []string{"https://etcd-1:2379/metrics", "https://etcd-2:2379/metrics"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getSortedTargets(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestCompareAndUpdateETCDTargetsFocused tests our key extracted function
func TestCompareAndUpdateETCDTargetsFocused(t *testing.T) {
	testCases := []struct {
		name              string
		existingTargets   string
		discoveredTargets []string
		expectUpdate      bool
	}{
		{
			name:              "No existing, new targets",
			existingTargets:   "",
			discoveredTargets: []string{"https://10.0.0.1:2379/metrics"},
			expectUpdate:      true,
		},
		{
			name:              "Same targets",
			existingTargets:   "https://10.0.0.1:2379/metrics",
			discoveredTargets: []string{"https://10.0.0.1:2379/metrics"},
			expectUpdate:      false,
		},
		{
			name:              "Different order, same content",
			existingTargets:   "https://10.0.0.2:2379/metrics,https://10.0.0.1:2379/metrics",
			discoveredTargets: []string{"https://10.0.0.1:2379/metrics", "https://10.0.0.2:2379/metrics"},
			expectUpdate:      false,
		},
		{
			name:              "Different targets",
			existingTargets:   "https://10.0.0.1:2379/metrics",
			discoveredTargets: []string{"https://10.0.0.2:2379/metrics"},
			expectUpdate:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a deployment with the specified targets
			deployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "test-namespace",
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name: constants.ContainerK8Sensor,
									Env: []corev1.EnvVar{
										{
											Name:  constants.EnvETCDTargets,
											Value: tc.existingTargets,
										},
									},
								},
							},
						},
					},
				},
			}

			logger := zap.New()
			needsUpdate := compareAndUpdateETCDTargets(deployment, tc.discoveredTargets, logger)

			assert.Equal(t, tc.expectUpdate, needsUpdate, "Update expectation should match")
		})
	}
}
