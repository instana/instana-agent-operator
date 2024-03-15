package helpers

import (
	"strings"

	corev1 "k8s.io/api/core/v1"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

const (
	ContainersInstanaIORegistry = "containers.instana.io"
)

type helpers struct {
	*instanav1.InstanaAgent
}

type Helpers interface {
	ServiceAccountName() string
	KeysSecretName() string
	TLSIsEnabled() bool
	TLSSecretName() string
	HeadlessServiceName() string
	K8sSensorResourcesName() string
	ContainersSecretName() string
	UseContainersSecret() bool
	ImagePullSecrets() []corev1.LocalObjectReference
}

func (h *helpers) serviceAccountNameDefault() string {
	switch pointer.DerefOrEmpty(h.Spec.ServiceAccountSpec.Create.Create) {
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
	return h.Spec.Agent.TlsSpec.SecretName != "" || (len(h.Spec.Agent.TlsSpec.Certificate) > 0 && len(h.Spec.Agent.TlsSpec.Key) > 0)
}

func (h *helpers) TLSSecretName() string {
	return optional.Of(h.Spec.Agent.TlsSpec.SecretName).GetOrDefault(h.Name + "-tls")
}

func (h *helpers) HeadlessServiceName() string {
	return h.Name + "-headless"
}

func (h *helpers) K8sSensorResourcesName() string {
	return h.Name + "-k8sensor"
}

func (h *helpers) ContainersSecretName() string {
	return h.Name + "-containers-instana-io"
}

func (h *helpers) UseContainersSecret() bool {
	// Explicitly using e.PullSecrets != nil instead of len(e.PullSecrets) == 0 since the original chart specified that
	// auto-generated secret shouldn't be used if the user explicitly provided an empty list of pull secrets
	// (original logic was to only use the generated secret if the registry matches AND the pullSecrets field was
	// omitted by the user). I don't understand why anyone would want this, but the original chart had comments
	// specifically mentioning that this was the desired behavior, so keeping it until someone says otherwise.
	return h.Spec.Agent.PullSecrets == nil && strings.HasPrefix(
		h.Spec.Agent.ImageSpec.Name,
		ContainersInstanaIORegistry,
	)
}

func (h *helpers) ImagePullSecrets() []corev1.LocalObjectReference {
	if h.UseContainersSecret() {
		return []corev1.LocalObjectReference{
			{
				Name: h.ContainersSecretName(),
			},
		}
	} else {
		return h.Spec.Agent.ExtendedImageSpec.PullSecrets
	}
}

func NewHelpers(agent *instanav1.InstanaAgent) Helpers {
	return &helpers{
		InstanaAgent: agent,
	}
}
