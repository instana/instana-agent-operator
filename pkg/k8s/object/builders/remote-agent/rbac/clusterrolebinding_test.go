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

package rbac

import (
	"testing"

	"github.com/instana/instana-agent-operator/mocks"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
)

func TestRemoteClusterRoleBindingBuilder_IsNamespaced_ComponentName(t *testing.T) {
	assertions := require.New(t)

	crb := NewClusterRoleBindingBuilder(nil)

	assertions.False(crb.IsNamespaced())
	assertions.Equal(constants.ComponentRemoteAgent, crb.ComponentName())
}

func TestRemoteClusterRoleBindingBuilder_Build(t *testing.T) {
	assertions := require.New(t)
	ctrl := gomock.NewController(t)

	sensorResourcesName := rand.String(10)
	namespace := rand.String(10)

	agent := &instanav1.RemoteAgent{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
	}

	expected := optional.Of[client.Object](
		&rbacv1.ClusterRoleBinding{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       "ClusterRoleBinding",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "remote-agent-" + sensorResourcesName,
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "remote-agent-" + sensorResourcesName,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      "remote-agent-" + sensorResourcesName,
					Namespace: namespace,
				},
			},
		},
	)

	helpers := mocks.NewMockHelpers(ctrl)
	helpers.EXPECT().ServiceAccountName().Times(1).Return(sensorResourcesName)

	crb := &clusterRoleBindingBuilder{
		RemoteAgent:   agent,
		RemoteHelpers: helpers,
	}

	actual := crb.Build()

	assertions.Equal(expected, actual)
}
