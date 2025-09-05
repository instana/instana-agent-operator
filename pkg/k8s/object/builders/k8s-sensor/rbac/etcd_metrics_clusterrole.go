/*
(c) Copyright IBM Corp. 2024
(c) Copyright Instana Inc.
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

type etcdMetricsClusterRoleBuilder struct {
	*instanav1.InstanaAgent
	helpers.Helpers
}

func (c *etcdMetricsClusterRoleBuilder) ComponentName() string {
	return constants.ComponentK8Sensor
}

func (c *etcdMetricsClusterRoleBuilder) IsNamespaced() bool {
	return false
}

func (c *etcdMetricsClusterRoleBuilder) Build() optional.Optional[client.Object] {
	return optional.Of[client.Object](
		&rbacv1.ClusterRole{
			TypeMeta: metav1.TypeMeta{
				APIVersion: rbacApiVersion,
				Kind:       roleKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: c.K8sSensorResourcesName() + "-etcd-metrics",
			},
			Rules: []rbacv1.PolicyRule{
				{
					NonResourceURLs: []string{"/metrics"},
					Verbs:           []string{"get"},
				},
			},
		},
	)
}

func NewEtcdMetricsClusterRoleBuilder(agent *instanav1.InstanaAgent) builder.ObjectBuilder {
	return &etcdMetricsClusterRoleBuilder{
		InstanaAgent: agent,
		Helpers:      helpers.NewHelpers(agent),
	}
}
