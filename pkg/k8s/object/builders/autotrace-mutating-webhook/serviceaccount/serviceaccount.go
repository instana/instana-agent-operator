/*
(c) Copyright IBM Corp. 2025
(c) Copyright Instana Inc. 2025
*/

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

const componentName = constants.ComponentAutoTraceWebhook

type serviceAccountBuilder struct {
	*instanav1.InstanaAgent
	helpers helpers.Helpers
}

func (d *serviceAccountBuilder) IsNamespaced() bool {
	return true
}

func (d *serviceAccountBuilder) ComponentName() string {
	return componentName
}

func (d *serviceAccountBuilder) Build() (res optional.Optional[client.Object]) {

	return optional.Of[client.Object](
		&corev1.ServiceAccount{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ServiceAccount",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      d.helpers.AutotraceWebhookResourcesName(),
				Namespace: d.helpers.AutotraceWebhookResourcesName(),
				//todo: add labels
			},
		},
	)
}

func NewServiceAccountBuilder(
	agent *instanav1.InstanaAgent,
) builder.ObjectBuilder {
	return &serviceAccountBuilder{
		InstanaAgent: agent,
		helpers:      helpers.NewHelpers(agent),
	}
}
