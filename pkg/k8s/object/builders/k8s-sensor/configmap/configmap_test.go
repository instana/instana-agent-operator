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

package configmap

import (
	"fmt"
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

func TestConfigMapBuilderIsNamespacedComponentName(t *testing.T) {
	assertions := require.New(t)

	cmb := NewConfigMapBuilder(nil, "", "", "")

	assertions.True(cmb.IsNamespaced())
	assertions.Equal(constants.ComponentK8Sensor, cmb.ComponentName())
}

func TestConfigMapBuilderBuild(t *testing.T) {
	assertions := require.New(t)
	ctrl := gomock.NewController(t)

	sensorResourcesName := rand.String(10)
	namespace := rand.String(10)

	endpointHost := rand.String(10)
	endpointPort := rand.String(10)

	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				EndpointHost: endpointHost,
				EndpointPort: endpointPort,
			},
		},
	}

	expected := optional.Of[client.Object](
		&corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      sensorResourcesName,
				Namespace: namespace,
			},
			Data: map[string]string{
				"backend": fmt.Sprintf("%s:%s", endpointHost, endpointPort),
			},
		},
	)

	helpers := mocks.NewMockHelpers(ctrl)
	helpers.EXPECT().K8sSensorResourcesName().Return(sensorResourcesName)

	cmb := &configMapBuilder{
		InstanaAgent: agent,
		Helpers:      helpers,
		endpointHost: endpointHost,
		endpointPort: endpointPort,
	}

	actual := cmb.Build()

	assertions.Equal(expected, actual)
}
