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
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/instana/instana-agent-operator/internal/mocks"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

func TestRoleBuilder_IsNamespaced_ComponentName(t *testing.T) {
	assertions := require.New(t)

	rb := NewRoleBuilder(nil)

	assertions.True(rb.IsNamespaced())
	assertions.Equal(constants.ComponentK8Sensor, rb.ComponentName())
}

func TestRoleBuilder_Build(t *testing.T) {
	assertions := require.New(t)

	sensorResourcesName := rand.String(10)

	expected := optional.Of[client.Object](
		&rbacv1.Role{
			TypeMeta: metav1.TypeMeta{
				APIVersion: rbacApiVersion,
				Kind:       "Role",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      sensorResourcesName + "-etcd-reader",
				Namespace: "kube-system",
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"services", "endpoints"},
					Verbs:     constants.ReaderVerbs(),
				},
			},
		},
	)

	helpers := &mocks.MockHelpers{}
	defer helpers.AssertExpectations(t)
	helpers.On("K8sSensorResourcesName").Return(sensorResourcesName).Once()

	rb := &roleBuilder{
		Helpers: helpers,
	}

	actual := rb.Build()

	assertions.Equal(expected, actual)
}