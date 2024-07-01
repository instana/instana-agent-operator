/*
(c) Copyright IBM Corp. 2024
(c) Copyright Instana Inc.

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

package status

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDaemonSetIsAvailable(t *testing.T) {
	for _, test := range []struct {
		name      string
		daemonset appsv1.DaemonSet
		expected  bool
	}{
		{
			name:      "Should return false when DaemonSet.Status is not present",
			daemonset: appsv1.DaemonSet{},
			expected:  false,
		},
		{
			name: "Should return false when DaemonSet.Generation is not DemonSetStatus.ObservedGeneration",
			daemonset: appsv1.DaemonSet{
				Status: appsv1.DaemonSetStatus{ObservedGeneration: int64(2)},
				ObjectMeta: metav1.ObjectMeta{
					Generation: int64(1),
				},
			},
			expected: false,
		},
		{
			name: "Should return false when DaemonSetStatus.NumberMisscheduled is not zero",
			daemonset: appsv1.DaemonSet{
				Status: appsv1.DaemonSetStatus{
					NumberMisscheduled: 2,
					ObservedGeneration: int64(1),
				},
				ObjectMeta: metav1.ObjectMeta{
					Generation: int64(1),
				},
			},
			expected: false,
		},
		{
			name: "Should return false when DaemonSetStatus.DesiredNumberScheduled does not match DaemonSetStatus.NumberAvailable",
			daemonset: appsv1.DaemonSet{
				Status: appsv1.DaemonSetStatus{
					DesiredNumberScheduled: 2,
					NumberAvailable:        1,
					ObservedGeneration:     int64(1),
				},
				ObjectMeta: metav1.ObjectMeta{
					Generation: int64(1),
				},
			},
			expected: false,
		},
		{
			name: "Should return false when DaemonSetStatus.DesiredNumberScheduled does not match DaemonSetStatus.UpdateNumberScheduled",
			daemonset: appsv1.DaemonSet{
				Status: appsv1.DaemonSetStatus{
					DesiredNumberScheduled: 2,
					NumberAvailable:        2,
					UpdatedNumberScheduled: 3,
					ObservedGeneration:     int64(1),
				},
				ObjectMeta: metav1.ObjectMeta{
					Generation: int64(1),
				},
			},
			expected: false,
		},
		{
			name: "Should return true when statuses match",
			daemonset: appsv1.DaemonSet{
				Status: appsv1.DaemonSetStatus{
					DesiredNumberScheduled: 2,
					NumberAvailable:        2,
					UpdatedNumberScheduled: 2,
					ObservedGeneration:     int64(1),
				},
				ObjectMeta: metav1.ObjectMeta{
					Generation: int64(1),
				},
			},
			expected: true,
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				require.New(t).Equal(daemonsetIsAvailable(test.daemonset), test.expected)
			},
		)
	}
}

func TestDeploymentHasMinimumAvailability(t *testing.T) {
	for _, test := range []struct {
		name                string
		deploymentCondition []appsv1.DeploymentCondition
		expected            bool
	}{
		{
			name:                "Should return true when DeploymentCondition with the type DeploymentAvailable and Status is ConditionTrue",
			deploymentCondition: []appsv1.DeploymentCondition{{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionTrue}},
			expected:            true,
		},
		{
			name:                "Should return false when DeploymentCondition with the type DeploymentAvailable and Status is ConditionFalse",
			deploymentCondition: []appsv1.DeploymentCondition{{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionFalse}},
			expected:            false,
		},
		{
			name:                "Should return false when DeploymentCondition type is not DeploymentProcessing or has not been populated yet",
			deploymentCondition: []appsv1.DeploymentCondition{{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionFalse}},
			expected:            false,
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				require.New(t).Equal(deploymentHasMinimumAvailability(deploymentConditionsAsMap(test.deploymentCondition)), test.expected)
			},
		)
	}
}

func TestDeploymentIsComplete(t *testing.T) {
	for _, test := range []struct {
		name                string
		deploymentCondition []appsv1.DeploymentCondition
		expected            bool
	}{
		{
			name:                "Should return true when DeploymentCondition with the type DeploymentProcessing has Status as ConditionTrue and Reason as NewReplicaSetAvailable",
			deploymentCondition: []appsv1.DeploymentCondition{{Type: appsv1.DeploymentProgressing, Status: corev1.ConditionTrue, Reason: "NewReplicaSetAvailable"}},
			expected:            true,
		},
		{
			name:                "Should return false when DeploymentCondition with the type DeploymentProcessing has Status as ConditionFalse even when Reason is NewReplicaSetAvailable",
			deploymentCondition: []appsv1.DeploymentCondition{{Type: appsv1.DeploymentProgressing, Status: corev1.ConditionFalse, Reason: "NewReplicaSetAvailable"}},
			expected:            false,
		},
		{
			name:                "Should return false when DeploymentCondition type is not DeploymentProcessing or has not been populated yet",
			deploymentCondition: []appsv1.DeploymentCondition{},
			expected:            false,
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				require.New(t).Equal(deploymentIsComplete(deploymentConditionsAsMap(test.deploymentCondition)), test.expected)
			},
		)
	}
}

func TestDeploymentIsAvailableAndComplete(t *testing.T) {
	for _, test := range []struct {
		name       string
		deployment appsv1.Deployment
		expected   bool
	}{
		{
			name: "Should return false when DeploymentStatus.ObservedGeneration is not Deployment.Generation",
			deployment: appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Generation: int64(1),
				},
				Status: appsv1.DeploymentStatus{
					ObservedGeneration: int64(2),
					Conditions: []appsv1.DeploymentCondition{
						{Type: appsv1.DeploymentProgressing, Status: corev1.ConditionTrue, Reason: "NewReplicaSetAvailable"},
						{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionTrue},
						{Type: appsv1.DeploymentReplicaFailure, Status: corev1.ConditionTrue},
					},
				},
			},
			expected: false,
		},
		{
			name: "Should return false when deployment does not have minimum availability",
			deployment: appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Generation: int64(1),
				},
				Status: appsv1.DeploymentStatus{
					ObservedGeneration: int64(1),
					Conditions: []appsv1.DeploymentCondition{
						{Type: appsv1.DeploymentProgressing, Status: corev1.ConditionFalse, Reason: "NewReplicaSetAvailable"},
						{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionTrue},
						{Type: appsv1.DeploymentReplicaFailure, Status: corev1.ConditionTrue},
					},
				},
			},
			expected: false,
		},
		{
			name: "Should return false when deployment contains replica failures",
			deployment: appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Generation: int64(1),
				},
				Status: appsv1.DeploymentStatus{
					ObservedGeneration: int64(1),
					Conditions: []appsv1.DeploymentCondition{
						{Type: appsv1.DeploymentProgressing, Status: corev1.ConditionTrue, Reason: "NewReplicaSetAvailable"},
						{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionTrue},
						{Type: appsv1.DeploymentReplicaFailure, Status: corev1.ConditionTrue},
					},
				},
			},
			expected: false,
		},
		{
			name: "Should return true when deployment is available and complete",
			deployment: appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Generation: int64(1),
				},
				Status: appsv1.DeploymentStatus{
					ObservedGeneration: int64(1),
					Conditions: []appsv1.DeploymentCondition{
						{Type: appsv1.DeploymentProgressing, Status: corev1.ConditionTrue, Reason: "NewReplicaSetAvailable"},
						{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionTrue},
						{Type: appsv1.DeploymentReplicaFailure, Status: corev1.ConditionFalse},
					},
				},
			},
			expected: true,
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				require.New(t).Equal(deploymentIsAvailableAndComplete(test.deployment), test.expected)
			},
		)
	}
}

func TestDeploymentHasReplicaFailures(t *testing.T) {
	for _, test := range []struct {
		name                string
		deploymentCondition []appsv1.DeploymentCondition
		expected            bool
	}{
		{
			name:                "Should return true when DeploymentCondition with the type DeploymentReplicaFailure has Status as ConditionTrue",
			deploymentCondition: []appsv1.DeploymentCondition{{Type: appsv1.DeploymentReplicaFailure, Status: corev1.ConditionTrue}},
			expected:            true,
		},
		{
			name:                "Should return false when DeploymentCondition does not have DeploymentReplicaFailure",
			deploymentCondition: []appsv1.DeploymentCondition{},
			expected:            false,
		},
		{
			name:                "Should return false when DeploymentCondition with the type DeploymentReplicaFailure has Status as ConditionFalse",
			deploymentCondition: []appsv1.DeploymentCondition{{Type: appsv1.DeploymentReplicaFailure, Status: corev1.ConditionFalse}},
			expected:            false,
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				require.New(t).Equal(deploymentHasReplicaFailures(deploymentConditionsAsMap(test.deploymentCondition)), test.expected)
			},
		)
	}
}

func TestTruncateMessageWhenLengthIsTooLong(t *testing.T) {
	assertions := require.New(t)

	var builder strings.Builder
	baseString := "This is a part of the long string. "
	for i := 0; i < 1000; i++ {
		builder.Write([]byte(baseString))
	}

	longString := builder.String()
	response := truncateMessage(longString)

	assertions.NotEqual(len(longString), 32768)
	assertions.Equal(len(response), 32768)
}
