package serviceaccount

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

type serviceAccountBuilder struct {
	*instanav1.InstanaAgent
	helpers.Helpers
}

func (s *serviceAccountBuilder) IsNamespaced() bool {
	return true
}

func (s *serviceAccountBuilder) ComponentName() string {
	return constants.ComponentInstanaAgent
}

func (s *serviceAccountBuilder) build() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.ServiceAccountName(),
			Namespace: s.Namespace,
		},
	}
}

func (s *serviceAccountBuilder) Build() optional.Optional[client.Object] {
	if s.Spec.ServiceAccountSpec.Create.Create {
		return optional.Of[client.Object](s.build())
	} else {
		return optional.Empty[client.Object]()
	}
}

func NewServiceAccountBuilder(agent *instanav1.InstanaAgent) builder.ObjectBuilder {
	return &serviceAccountBuilder{
		InstanaAgent: agent,
		Helpers:      helpers.NewHelpers(agent),
	}
}
