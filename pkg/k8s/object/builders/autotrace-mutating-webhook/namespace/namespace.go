/*
(c) Copyright IBM Corp. 2025
(c) Copyright Instana Inc. 2025
*/

package namespace

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

type namespaceBuilder struct {
	*instanav1.InstanaAgent
	helpers helpers.Helpers
}

func (ns *namespaceBuilder) IsNamespaced() bool {
	return false
}

func (ns *namespaceBuilder) ComponentName() string {
	return ns.helpers.AutotraceWebhookResourcesName()
}

func (ns *namespaceBuilder) Build() (res optional.Optional[client.Object]) {

	return optional.Of[client.Object](
		&corev1.Namespace{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Namespace",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: ns.ComponentName(),
			},
		},
	)

}

func NewNamespaceBuilder(
	agent *instanav1.InstanaAgent,
) builder.ObjectBuilder {
	return &namespaceBuilder{
		InstanaAgent: agent,
		helpers:      helpers.NewHelpers(agent),
	}
}
