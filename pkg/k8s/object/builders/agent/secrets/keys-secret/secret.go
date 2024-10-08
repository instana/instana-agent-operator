/*
(c) Copyright IBM Corp. 2024
*/

package keys_secret

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	backends "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/backends"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

type secretBuilder struct {
	*instanav1.InstanaAgent
	backends []backends.K8SensorBackend
}

func NewSecretBuilder(agent *instanav1.InstanaAgent, backends []backends.K8SensorBackend) builder.ObjectBuilder {
	return &secretBuilder{
		InstanaAgent: agent,
		backends:     backends,
	}
}

func (s *secretBuilder) IsNamespaced() bool {
	return true
}

func (s *secretBuilder) ComponentName() string {
	return constants.ComponentInstanaAgent
}

func (s *secretBuilder) Build() optional.Optional[client.Object] {
	switch s.Spec.Agent.KeysSecret {
	case "":
		return optional.Of[client.Object](s.build())
	default:
		return optional.Empty[client.Object]()
	}
}

func (s *secretBuilder) build() *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.Name,
			Namespace: s.Namespace,
		},
		Data: s.getData(),
		Type: corev1.SecretTypeOpaque,
	}
}

func (s *secretBuilder) getData() map[string][]byte {
	data := make(map[string][]byte, len(s.backends)+1)

	optional.Of(s.Spec.Agent.DownloadKey).IfPresent(
		func(downloadKey string) {
			data[constants.DownloadKey] = []byte(downloadKey)
		},
	)

	for _, backend := range s.backends {
		optional.Of(backend.EndpointKey).IfPresent(
			func(key string) {
				data[constants.AgentKey+backend.ResourceSuffix] = []byte(key)
			},
		)
	}

	return data
}
