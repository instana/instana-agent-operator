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

package serviceaccount

import (
	"testing"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/mocks"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

func TestServiceAccountBuilder_IsNamespaced_ComponentName(t *testing.T) {
	assertions := require.New(t)

	sb := NewServiceAccountBuilder(nil)

	assertions.True(sb.IsNamespaced())
	assertions.Equal(constants.ComponentK8Sensor, sb.ComponentName())
}

func TestServiceAccountBuilder_Build(t *testing.T) {
	assertions := require.New(t)
	ctrl := gomock.NewController(t)

	sensorResourcesName := rand.String(10)
	namespace := rand.String(10)

	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
	}

	expected := optional.Of[client.Object](
		&corev1.ServiceAccount{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ServiceAccount",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      sensorResourcesName,
				Namespace: namespace,
			},
		},
	)

	helpers := mocks.NewMockHelpers(ctrl)
	helpers.EXPECT().K8sSensorResourcesName().Return(sensorResourcesName)

	sb := &serviceAccountBuilder{
		InstanaAgent: agent,
		Helpers:      helpers,
	}

	actual := sb.Build()

	assertions.Equal(expected, actual)
}
