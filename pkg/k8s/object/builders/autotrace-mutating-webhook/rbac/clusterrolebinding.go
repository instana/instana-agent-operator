/*
(c) Copyright IBM Corp. 2025
(c) Copyright Instana Inc. 2025
*/

package rbac

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

type clusterRoleBindingBuilder struct {
	*instanav1.InstanaAgent
	helpers helpers.Helpers
}

func (crb *clusterRoleBindingBuilder) IsNamespaced() bool {
	return false
}

func (crb *clusterRoleBindingBuilder) ComponentName() string {
	return crb.helpers.AutotraceWebhookResourcesName() + "-binding"
}

func (crb *clusterRoleBindingBuilder) Build() (res optional.Optional[client.Object]) {

	return optional.Of[client.Object](
		&rbacv1.ClusterRoleBinding{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       "ClusterRoleBinding",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: crb.ComponentName(),
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     crb.helpers.AutotraceWebhookResourcesName() + "-clusterrole",
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      rbacv1.ServiceAccountKind,
					Name:      crb.helpers.AutotraceWebhookResourcesName(),
					Namespace: crb.Namespace,
				},
			},
		},
	)
}

func NewClusterRoleBindingBuilder(
	agent *instanav1.InstanaAgent,
) builder.ObjectBuilder {
	return &clusterRoleBindingBuilder{
		InstanaAgent: agent,
		helpers:      helpers.NewHelpers(agent),
	}
}
