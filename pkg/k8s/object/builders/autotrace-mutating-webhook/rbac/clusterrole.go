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
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

const componentName = constants.ComponentAutoTraceWebhook

type clusterRoleBuilder struct {
	*instanav1.InstanaAgent
	helpers helpers.Helpers
}

func (d *clusterRoleBuilder) IsNamespaced() bool {
	return false
}

func (d *clusterRoleBuilder) ComponentName() string {
	return componentName
}

func (d *clusterRoleBuilder) Build() (res optional.Optional[client.Object]) {

	return optional.Of[client.Object](
		&rbacv1.ClusterRole{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       "ClusterRole",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: d.helpers.AutotraceWebhookResourcesName() + "-clusterrole",
				//todo: add labels
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
