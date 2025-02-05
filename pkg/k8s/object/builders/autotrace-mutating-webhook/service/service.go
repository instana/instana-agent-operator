/*
(c) Copyright IBM Corp. 2025
(c) Copyright Instana Inc. 2025
*/

package service

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

type serviceBuilder struct {
	*instanav1.InstanaAgent
	helpers helpers.Helpers
}

func (s *serviceBuilder) IsNamespaced() bool {
	return true
}

func (s *serviceBuilder) ComponentName() string {
	return s.helpers.AutotraceWebhookResourcesName()
}

func (s *serviceBuilder) Build() (res optional.Optional[client.Object]) {

	return optional.Of[client.Object](
		&corev1.Service{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Service",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      s.ComponentName(),
				Namespace: s.Namespace,
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					"app.kubernetes.io/instance": s.ComponentName(),
				},
				Ports: []corev1.ServicePort{
					{
						Name:       "https",
						Protocol:   corev1.ProtocolTCP,
						Port:       42650,
						TargetPort: intstr.FromInt(42650),
					},
				},
			},
		},
	)
}

func NewServiceBuilder(
	agent *instanav1.InstanaAgent,
) builder.ObjectBuilder {
	return &serviceBuilder{
		InstanaAgent: agent,
		helpers:      helpers.NewHelpers(agent),
	}
}
