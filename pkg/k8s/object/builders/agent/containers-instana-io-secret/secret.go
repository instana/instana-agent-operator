package containers_instana_io_secret

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/json_or_die"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

// TODO: Test

type secretBuilder struct {
	*instanav1.InstanaAgent

	helpers.Helpers
	json_or_die.JsonOrDieMarshaler[*DockerConfigJson]
}

func (s *secretBuilder) IsNamespaced() bool {
	return true
}

func (s *secretBuilder) ComponentName() string {
	return constants.ComponentInstanaAgent
}

func (s *secretBuilder) buildDockerConfigJson() []byte {
	password := optional.Of(s.Spec.Agent.DownloadKey).GetOrDefault(s.Spec.Agent.Key)
	auth := fmt.Sprintf("_:%s", password)

	json := DockerConfigJson{
		Auths: map[string]DockerConfigAuth{
			helpers.ContainersInstanaIORegistry: {
				Auth: []byte(auth),
			},
		},
	}

	return s.MarshalOrDie(&json)
}

func (s *secretBuilder) build() *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.ContainersSecretName(),
			Namespace: s.Namespace,
		},
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: s.buildDockerConfigJson(),
		},
		Type: corev1.SecretTypeDockerConfigJson,
	}
}

func (s *secretBuilder) Build() optional.Optional[client.Object] {
	switch s.UseContainersSecret() {
	case true:
		return optional.Of[client.Object](s.build())
	default:
		return optional.Empty[client.Object]()
	}
}

func NewSecretBuilder(agent *instanav1.InstanaAgent) builder.ObjectBuilder {
	return &secretBuilder{
		InstanaAgent:       agent,
		Helpers:            helpers.NewHelpers(agent),
		JsonOrDieMarshaler: json_or_die.NewJsonOrDie[DockerConfigJson](),
	}
}
