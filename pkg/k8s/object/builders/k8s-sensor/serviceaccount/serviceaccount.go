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
	return constants.ComponentK8Sensor
}

func (s *serviceAccountBuilder) Build() optional.Optional[client.Object] {
	return optional.Of[client.Object](
		&corev1.ServiceAccount{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ServiceAccount",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      s.K8sSensorResourcesName(),
				Namespace: s.Namespace,
			},
		},
	)
}

func NewServiceAccountBuilder(agent *instanav1.InstanaAgent) builder.ObjectBuilder {
	return &serviceAccountBuilder{
		InstanaAgent: agent,
		Helpers:      helpers.NewHelpers(agent),
	}
}
