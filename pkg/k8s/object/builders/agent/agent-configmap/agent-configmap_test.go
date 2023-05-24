package agent_configmap

import (
	"testing"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

func TestAgentConfigMapBuilder_Build(t *testing.T) {
	assertions := require.New(t)

	builder := NewAgentConfigMapBuilder(
		&instanav1.InstanaAgent{
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
		},
	)

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
				"configuration-opentelemetry.yaml":             "grpc: {}\n",
				"configuration-prometheus-remote-write.yaml":   "com.instana.plugin.prometheus:\n    remote_write:\n        enabled: true\n",
				"configuration-disable-kubernetes-sensor.yaml": "com.instana.plugin.kubernetes:\n    enabled: false\n",
				"additional-backend-2":                         "host=eoijsdlkjf\nport=goieoijsdofj\nkey=eoisdljsdlkfj\nprotocol=HTTP/2\nproxy.type=HTTP\nproxy.host=weoisdoijsdg\nproxy.port=lksdlkjsdglkjsd\nproxy.user=peoijsadglkj\nproxy.password=relksdlkj\nproxyUseDNS=true",
				"additional-backend-3":                         "host=glknsdlknmdsflk\nport=lgslkjsdfoieoiljsdf\nkey=sdlkjsadofjpoej\nprotocol=HTTP/2\nproxy.type=HTTP\nproxy.host=weoisdoijsdg\nproxy.port=lksdlkjsdglkjsd\nproxy.user=peoijsadglkj\nproxy.password=relksdlkj\nproxyUseDNS=true",
			},
		},
	)

	assertions.Equal(expected, actual)
}
