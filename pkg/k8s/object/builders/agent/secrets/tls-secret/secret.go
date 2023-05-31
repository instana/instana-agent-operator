package tls_secret

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

// TODO: Test

type secretBuilder struct {
	*instanav1.InstanaAgent

	helpers.Helpers
}

func (s *secretBuilder) IsNamespaced() bool {
	return true
}

func (s *secretBuilder) ComponentName() string {
	return constants.ComponentInstanaAgent
}

func (s *secretBuilder) build() *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: s.TLSSecretName(),
		},
		Data: map[string][]byte{
			corev1.TLSCertKey:       s.Spec.Agent.TlsSpec.Certificate,
			corev1.TLSPrivateKeyKey: s.Spec.Agent.TlsSpec.Key,
		},
		Type: corev1.SecretTypeTLS,
	}
}

func (s *secretBuilder) Build() optional.Optional[client.Object] {
	switch tls := s.Spec.Agent.TlsSpec; tls.SecretName == "" && len(tls.Key) > 0 && len(tls.Certificate) > 0 {
	case true:
		return optional.Of[client.Object](s.build())
	default:
		return optional.Empty[client.Object]()
	}
}

func NewSecretBuilder(agent *instanav1.InstanaAgent) builder.ObjectBuilder {
	return &secretBuilder{
		InstanaAgent: agent,

		Helpers: helpers.NewHelpers(agent),
	}
}
