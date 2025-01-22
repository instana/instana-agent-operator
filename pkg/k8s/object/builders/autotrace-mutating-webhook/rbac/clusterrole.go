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

type clusterRoleBuilder struct {
	*instanav1.InstanaAgent
	helpers helpers.Helpers
}

func (cr *clusterRoleBuilder) IsNamespaced() bool {
	return false
}

func (cr *clusterRoleBuilder) ComponentName() string {
	return cr.helpers.AutotraceWebhookResourcesName() + "-clusterrole"
}

func (cr *clusterRoleBuilder) Build() (res optional.Optional[client.Object]) {

	return optional.Of[client.Object](
		&rbacv1.ClusterRole{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       "ClusterRole",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: cr.ComponentName(),
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{
						"secrets",
						"configmaps",
						"namespaces",
					},
					Verbs: []string{
						"get",
						"list",
						"watch",
						"create",
					},
				},
			},
		},
	)
}

func NewClusterRoleBuilder(
	agent *instanav1.InstanaAgent,
) builder.ObjectBuilder {
	return &clusterRoleBuilder{
		InstanaAgent: agent,
		helpers:      helpers.NewHelpers(agent),
	}
}
