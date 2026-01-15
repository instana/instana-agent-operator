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

package rbac

import (
	"testing"

	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

func TestAggregatedClusterRoleBuilder_IsNamespaced_ComponentName(t *testing.T) {
	assertions := require.New(t)

	cb := NewAggregatedClusterRoleBuilder(nil)

	assertions.False(cb.IsNamespaced())
	assertions.Equal(constants.ComponentK8Sensor, cb.ComponentName())
}

func TestAggregatedClusterRoleBuilder_Build(t *testing.T) {
	assertions := require.New(t)

	expected := optional.Of[client.Object](
		&rbacv1.ClusterRole{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       "ClusterRole",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "instana-crmon-aggregated",
			},
			AggregationRule: &rbacv1.AggregationRule{
				ClusterRoleSelectors: []metav1.LabelSelector{
					{
						MatchLabels: map[string]string{"instana.com/crmon": "true"},
					},
				},
			},
		},
	)

	cb := &aggregatedClusterRoleBuilder{}

	actual := cb.Build()

	assertions.Equal(expected, actual)
}
