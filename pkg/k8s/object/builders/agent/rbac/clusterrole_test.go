/*
(c) Copyright IBM Corp. 2025
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

package rbac

import (
	"testing"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/instana/instana-agent-operator/mocks"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

func TestClusterRoleBuilder_IsNamespaced_ComponentName(t *testing.T) {
	assertions := require.New(t)

	cb := NewClusterRoleBuilder(nil)

	assertions.False(cb.IsNamespaced())
	assertions.Equal(constants.ComponentInstanaAgent, cb.ComponentName())
}

func TestClusterRoleBuilder_Build(t *testing.T) {
	assertions := require.New(t)
	ctrl := gomock.NewController(t)

	sensorResourcesName := rand.String(10)

	expected := optional.Of[client.Object](
		&rbacv1.ClusterRole{
			TypeMeta: metav1.TypeMeta{
				APIVersion: rbacApiVersion,
				Kind:       roleKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: sensorResourcesName,
			},
			Rules: []rbacv1.PolicyRule{
				{
					NonResourceURLs: []string{
						"/version",
						"/healthz",
						"/metrics",
						"/stats/summary",
						"/metrics/cadvisor",
					},
					Verbs:     []string{"get"},
					APIGroups: []string{},
					Resources: []string{},
				},
				{
					APIGroups: []string{""},
					Resources: []string{
						"nodes",
						"nodes/stats",
						"nodes/metrics",
						"pods",
					},
					Verbs: constants.ReaderVerbs(),
				},
				{
					APIGroups:     []string{"security.openshift.io"},
					ResourceNames: []string{"privileged"},
					Resources: []string{
						"securitycontextconstraints",
					},
					Verbs: []string{"use"},
				},
				{
					APIGroups: []string{"policy"},
					Resources: []string{
						"podsecuritypolicies",
					},
					Verbs: []string{"use"},
				},
			},
		},
	)

	helpers := mocks.NewMockHelpers(ctrl)
	helpers.EXPECT().ServiceAccountName().Times(1).Return(sensorResourcesName)

	cb := &clusterRoleBuilder{
		Helpers: helpers,
	}

	actual := cb.Build()

	assertions.Equal(expected, actual)
}
