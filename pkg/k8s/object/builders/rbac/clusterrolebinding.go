package rbac

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/helpers"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

type clusterRoleBindingBuilder struct {
	*instanav1.InstanaAgent
	helpers.Helpers
}

func (c *clusterRoleBindingBuilder) IsNamespaced() bool {
	return false
}

func (c *clusterRoleBindingBuilder) ComponentName() string {
	return constants.ComponentK8Sensor
}

func (c *clusterRoleBindingBuilder) Build() optional.Optional[client.Object] {
	return optional.Of[client.Object](
		&rbacv1.ClusterRoleBinding{
			TypeMeta: metav1.TypeMeta{
				APIVersion: rbacApiVersion,
				Kind:       "ClusterRoleBinding",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: c.K8sSensorResourcesName(),
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacApiGroup,
				Kind:     roleKind,
				Name:     c.K8sSensorResourcesName(),
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      subjectKind,
					Name:      c.K8sSensorResourcesName(),
					Namespace: c.Namespace,
				},
			},
		},
	)
}

func NewClusterRoleBindingBuilder(agent *instanav1.InstanaAgent) builder.ObjectBuilder {
	return &clusterRoleBindingBuilder{
		InstanaAgent: agent,
		Helpers:      helpers.NewHelpers(agent),
	}
}
