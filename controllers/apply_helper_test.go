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

func TestCompareAndUpdateETCDTargets(t *testing.T) {
	// Test cases
	testCases := []struct {
		name              string
		existingTargets   string
		discoveredTargets []string
		expectUpdate      bool
		description       string
	}{
		{
			name:              "No existing targets, new targets discovered",
			existingTargets:   "",
			discoveredTargets: []string{"https://10.0.0.1:2379/metrics", "https://10.0.0.2:2379/metrics"},
			expectUpdate:      true,
			description:       "Should need update when no existing targets and new targets are discovered",
		},
		{
			name:              "Same targets, same order",
			existingTargets:   "https://10.0.0.1:2379/metrics,https://10.0.0.2:2379/metrics",
			discoveredTargets: []string{"https://10.0.0.1:2379/metrics", "https://10.0.0.2:2379/metrics"},
			expectUpdate:      false,
			description:       "Should not need update when targets are identical",
		},
		{
			name:              "Same targets, different order",
			existingTargets:   "https://10.0.0.2:2379/metrics,https://10.0.0.1:2379/metrics",
			discoveredTargets: []string{"https://10.0.0.1:2379/metrics", "https://10.0.0.2:2379/metrics"},
			expectUpdate:      false,
			description: "Should not need update when targets are same but in different order " +
				"(sorting should handle this)",
		},
		{
			name:              "Different targets",
			existingTargets:   "https://10.0.0.1:2379/metrics,https://10.0.0.2:2379/metrics",
			discoveredTargets: []string{"https://10.0.0.3:2379/metrics", "https://10.0.0.4:2379/metrics"},
			expectUpdate:      true,
			description:       "Should need update when targets are completely different",
		},
		{
			name:              "Additional target discovered",
			existingTargets:   "https://10.0.0.1:2379/metrics",
			discoveredTargets: []string{"https://10.0.0.1:2379/metrics", "https://10.0.0.2:2379/metrics"},
			expectUpdate:      true,
			description:       "Should need update when additional targets are discovered",
		},
		{
			name:              "Target removed",
			existingTargets:   "https://10.0.0.1:2379/metrics,https://10.0.0.2:2379/metrics",
			discoveredTargets: []string{"https://10.0.0.1:2379/metrics"},
			expectUpdate:      true,
			description:       "Should need update when targets are removed",
		},
		{
			name:              "Empty discovered targets",
			existingTargets:   "https://10.0.0.1:2379/metrics",
			discoveredTargets: []string{},
			expectUpdate:      true,
			description:       "Should need update when no targets are discovered but existing targets exist",
		},
		{
			name:              "Both empty",
			existingTargets:   "",
			discoveredTargets: []string{},
			expectUpdate:      false,
			description:       "Should not need update when both existing and discovered targets are empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a deployment with the specified existing targets
			deployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "k8s-sensor",
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

			// Test the function
			logger := zap.New()
			needsUpdate := compareAndUpdateETCDTargets(deployment, tc.discoveredTargets, logger)

			// Verify the result
			assert.Equal(t, tc.expectUpdate, needsUpdate, tc.description)
		})
	}
}

func TestCompareAndUpdateETCDTargetsWithoutK8SensorContainer(t *testing.T) {
	// Test case where deployment doesn't have k8s-sensor container
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "other-deployment",
			Namespace: "test-namespace",
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "other-container",
							Env: []corev1.EnvVar{
								{
									Name:  "OTHER_ENV",
									Value: "other-value",
								},
							},
						},
					},
				},
			},
		},
	}

	discoveredTargets := []string{"https://10.0.0.1:2379/metrics"}
	logger := zap.New()
	needsUpdate := compareAndUpdateETCDTargets(deployment, discoveredTargets, logger)

	// Should need update because no existing targets found (empty string) vs discovered targets
	assert.True(t, needsUpdate, "Should need update when k8s-sensor container is not found")
}

func TestCompareAndUpdateETCDTargetsWithoutETCDTargetsEnv(t *testing.T) {
	// Test case where k8s-sensor container exists but doesn't have ETCD_TARGETS env var
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "k8s-sensor",
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
									Name:  "OTHER_ENV",
									Value: "other-value",
								},
							},
						},
					},
				},
			},
		},
	}

	discoveredTargets := []string{"https://10.0.0.1:2379/metrics"}
	logger := zap.New()
	needsUpdate := compareAndUpdateETCDTargets(deployment, discoveredTargets, logger)

	// Should need update because no ETCD_TARGETS env var found (empty string) vs discovered targets
	assert.True(t, needsUpdate, "Should need update when ETCD_TARGETS env var is not found")
}
