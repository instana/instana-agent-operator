package helpers

import (
	v1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

type helpers struct {
	*v1.InstanaAgent
}

type Helpers interface {
	ServiceAccountName() string
	KeysSecretName() string
	TLSIsEnabled() bool
	TLSSecretName() string
}

func (h *helpers) serviceAccountNameDefault() string {
	switch h.Spec.ServiceAccountSpec.Create.Create {
	case true:
		return h.Name
	default:
		return "default"
	}
}

func (h *helpers) ServiceAccountName() string {
	return optional.Of(h.Spec.ServiceAccountSpec.Name.Name).GetOrDefault(h.serviceAccountNameDefault())
}

func (h *helpers) KeysSecretName() string {
	return optional.Of(h.Spec.Agent.KeysSecret).GetOrDefault(h.Name)
}

func (h *helpers) TLSIsEnabled() bool {
	return h.Spec.Agent.TlsSpec.SecretName != "" || (h.Spec.Agent.TlsSpec.Certificate != "" && h.Spec.Agent.TlsSpec.Key != "")
}

func (h *helpers) TLSSecretName() string {
	return optional.Of(h.Spec.Agent.TlsSpec.SecretName).GetOrDefault(h.Name + "-tls")
} // TODO: test

func NewHelpers(agent *v1.InstanaAgent) Helpers {
	return &helpers{
		InstanaAgent: agent,
	}
}
