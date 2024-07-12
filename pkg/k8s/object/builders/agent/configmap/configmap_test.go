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
	"testing"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/mocks"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

func TestAgentConfigMapBuilderBuild(t *testing.T) {
	assertions := require.New(t)
	ctrl := gomock.NewController(t)

	agentCm := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "llsdfoije",
			Namespace: "glkdsoijeijsd",
		},
		Spec: instanav1.InstanaAgentSpec{
			Cluster: instanav1.Name{
				Name: "eoisdgoijds",
			},
			Agent: instanav1.BaseAgentSpec{
				ConfigurationYaml: "riosoidoijdsg",
				ProxyHost:         "weoisdoijsdg",
				ProxyPort:         "lksdlkjsdglkjsd",
				ProxyUser:         "peoijsadglkj",
				ProxyPassword:     "relksdlkj",
				ProxyUseDNS:       true,
				AdditionalBackends: []instanav1.BackendSpec{
					{
						EndpointHost: "eoijsdlkjf",
						EndpointPort: "goieoijsdofj",
						Key:          "eoisdljsdlkfj",
					},
					{
						EndpointHost: "glknsdlknmdsflk",
						EndpointPort: "lgslkjsdfoieoiljsdf",
						Key:          "sdlkjsadofjpoej",
					},
				},
			},
			OpenTelemetry: instanav1.OpenTelemetry{
				GRPC: &instanav1.Enabled{},
			},
			Prometheus: instanav1.Prometheus{
				RemoteWrite: instanav1.Enabled{
					Enabled: pointer.To(true),
				},
			},
		},
	}

	statusManager := mocks.NewMockAgentStatusManager(ctrl)
	statusManager.EXPECT().SetAgentConfigMap(gomock.Eq(client.ObjectKeyFromObject(agentCm)))

	builder := NewConfigMapBuilder(agentCm, statusManager)

	actual := builder.Build()

	expected := optional.Of[client.Object](
		&v1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "llsdfoije",
				Namespace: "glkdsoijeijsd",
			},
			Data: map[string]string{
				"cluster_name":                                 "eoisdgoijds",
				"configuration.yaml":                           "riosoidoijdsg",
				"configuration-opentelemetry.yaml":             "com.instana.plugin.opentelemetry:\n    grpc: {}\n",
				"configuration-prometheus-remote-write.yaml":   "com.instana.plugin.prometheus:\n    remote_write:\n        enabled: true\n",
				"configuration-disable-kubernetes-sensor.yaml": "com.instana.plugin.kubernetes:\n    enabled: false\n",
				"additional-backend-2":                         "host=eoijsdlkjf\nport=goieoijsdofj\nkey=eoisdljsdlkfj\nprotocol=HTTP/2\nproxy.type=HTTP\nproxy.host=weoisdoijsdg\nproxy.port=lksdlkjsdglkjsd\nproxy.user=peoijsadglkj\nproxy.password=relksdlkj\nproxyUseDNS=true",
				"additional-backend-3":                         "host=glknsdlknmdsflk\nport=lgslkjsdfoieoiljsdf\nkey=sdlkjsadofjpoej\nprotocol=HTTP/2\nproxy.type=HTTP\nproxy.host=weoisdoijsdg\nproxy.port=lksdlkjsdglkjsd\nproxy.user=peoijsadglkj\nproxy.password=relksdlkj\nproxyUseDNS=true",
			},
		},
	)

	assertions.Equal(expected, actual)
}
