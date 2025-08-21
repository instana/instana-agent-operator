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

package poddisruptionbudget

import (
	"testing"

	"github.com/stretchr/testify/assert"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/internal/mocks"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

func TestPodDisruptionBudgetBuilderIsNamespacedComponentName(t *testing.T) {
	assertions := assert.New(t)

	pdbBuilder := NewPodDisruptionBudgetBuilder(nil)

	assertions.True(pdbBuilder.IsNamespaced())
	assertions.Equal(constants.ComponentK8Sensor, pdbBuilder.ComponentName())
}

func TestPodDisruptionBudgetBuilderBuild(t *testing.T) {
	agentName := rand.String(10)
	agentNamespace := rand.String(10)
	expectedPdbName := rand.String(10)
	expectedMatchLabels := map[string]string{
		rand.String(10): rand.String(10),
		rand.String(10): rand.String(10),
		rand.String(10): rand.String(10),
	}
	numReplicas := rand.Int63nRange(2, 1000)

	for _, test := range []struct {
		name     string
		agent    *instanav1.InstanaAgent
		expected optional.Optional[client.Object]
	}{
		{
			name:     "pdb_enablement_unset",
			agent:    &instanav1.InstanaAgent{},
			expected: optional.Empty[client.Object](),
		},
		{
			name: "pdb_explicitly_disabled",
			agent: &instanav1.InstanaAgent{
				Spec: instanav1.InstanaAgentSpec{
					K8sSensor: instanav1.K8sSpec{
						PodDisruptionBudget: instanav1.Enabled{Enabled: pointer.To(false)},
					},
				},
			},
			expected: optional.Empty[client.Object](),
		},
		{
			name: "pdb_enabled_but_replicas_is_zero",
			agent: &instanav1.InstanaAgent{
				Spec: instanav1.InstanaAgentSpec{
					K8sSensor: instanav1.K8sSpec{
						PodDisruptionBudget: instanav1.Enabled{Enabled: pointer.To(true)},
					},
				},
			},
			expected: optional.Empty[client.Object](),
		},
		{
			name: "pdb_enabled_but_replicas_is_one",
			agent: &instanav1.InstanaAgent{
				Spec: instanav1.InstanaAgentSpec{
					K8sSensor: instanav1.K8sSpec{
						PodDisruptionBudget: instanav1.Enabled{Enabled: pointer.To(true)},
						DeploymentSpec:      instanav1.KubernetesDeploymentSpec{Replicas: 1},
					},
				},
			},
			expected: optional.Empty[client.Object](),
		},
		{
			name: "pdb_enabled_and_replicas_at_greater_than_one",
			agent: &instanav1.InstanaAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      agentName,
					Namespace: agentNamespace,
				},
				Spec: instanav1.InstanaAgentSpec{
					K8sSensor: instanav1.K8sSpec{
						DeploymentSpec:      instanav1.KubernetesDeploymentSpec{Replicas: int(numReplicas)},
						PodDisruptionBudget: instanav1.Enabled{Enabled: pointer.To(true)},
					},
				},
			},
			expected: optional.Of[client.Object](
				&policyv1.PodDisruptionBudget{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "policy/v1",
						Kind:       "PodDisruptionBudget",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      expectedPdbName,
						Namespace: agentNamespace,
					},
					Spec: policyv1.PodDisruptionBudgetSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: expectedMatchLabels,
						},
						MinAvailable: pointer.To(intstr.FromInt32(int32(numReplicas) - 1)),
					},
				},
			),
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				helpers := &mocks.MockHelpers{}
				defer helpers.AssertExpectations(t)
				podSelectorGen := &mocks.MockPodSelectorLabelGenerator{}
				defer podSelectorGen.AssertExpectations(t)

				pdbBuilder := &podDisruptionBudgetBuilder{
					InstanaAgent:              test.agent,
					Helpers:                   helpers,
					PodSelectorLabelGenerator: podSelectorGen,
				}

				test.expected.IfPresent(
					func(_ client.Object) {
						helpers.On("K8sSensorResourcesName").Return(expectedPdbName)
						podSelectorGen.On("GetPodSelectorLabels").Return(expectedMatchLabels)
					},
				)

				actual := pdbBuilder.Build()
				assert.Equal(t, test.expected, actual)
			},
		)
	}
}
