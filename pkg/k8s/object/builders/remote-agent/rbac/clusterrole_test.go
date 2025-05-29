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

func TestRemoteClusterRoleBuilder_IsNamespaced_ComponentName(t *testing.T) {
	assertions := require.New(t)

	cb := NewClusterRoleBuilder(nil)

	assertions.False(cb.IsNamespaced())
	assertions.Equal(constants.ComponentRemoteAgent, cb.ComponentName())
}

func TestRemoteClusterRoleBuilder_Build(t *testing.T) {
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
				Name: "remote-agent-" + sensorResourcesName,
			},
		},
	)

	helpers := mocks.NewMockRemoteHelpers(ctrl)
	helpers.EXPECT().ServiceAccountName().Times(1).Return(sensorResourcesName)

	cb := &clusterRoleBuilder{
		RemoteHelpers: helpers,
	}

	actual := cb.Build()

	assertions.Equal(expected, actual)
}
