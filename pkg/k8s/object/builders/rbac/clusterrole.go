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

func readerVerbs() []string {
	return []string{"get", "list", "watch"}
}

// TODO: Test

// TODO: constructing based on permissions for external k8s sensor for now

type clusterRoleBuilder struct {
	*instanav1.InstanaAgent
	helpers.Helpers
}

func (c *clusterRoleBuilder) ComponentName() string {
	return constants.ComponentK8Sensor
}

func (c *clusterRoleBuilder) IsNamespaced() bool {
	return false
}

func (c *clusterRoleBuilder) build() client.Object {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.K8sSensorResourcesName(),
			Namespace: c.Namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				NonResourceURLs: []string{"/version", "/healthz"},
				Verbs:           []string{"get"},
				APIGroups:       []string{},
				Resources:       []string{},
			},
			{
				APIGroups: []string{"extensions"},
				Resources: []string{"deployments", "replicasets", "ingresses"},
				Verbs:     readerVerbs(),
			},
			{
				APIGroups: []string{""},
				Resources: []string{
					"configmaps",
					"events",
					"services",
					"endpoints",
					"namespaces",
					"nodes",
					"pods",
					"pods/log",
					"replicationcontrollers",
					"resourcequotas",
					"persistentvolumes",
					"persistentvolumeclaims",
				},
				Verbs: readerVerbs(),
			},
			{
				APIGroups: []string{"apps"},
				Resources: []string{"daemonsets", "deployments", "replicasets", "statefulsets"},
				Verbs:     readerVerbs(),
			},
			{
				APIGroups: []string{"batch"},
				Resources: []string{"cronjobs", "jobs"},
				Verbs:     readerVerbs(),
			},
			{
				APIGroups: []string{"networking.k8s.io"},
				Resources: []string{"ingresses"},
				Verbs:     readerVerbs(),
			},
			{
				APIGroups: []string{"autoscaling"},
				Resources: []string{"horizontalpodautoscalers"},
				Verbs:     readerVerbs(),
			},
			{
				APIGroups: []string{"apps.openshift.io"},
				Resources: []string{"deploymentconfigs"},
				Verbs:     readerVerbs(),
			},
			{
				APIGroups:     []string{"security.openshift.io"},
				ResourceNames: []string{"privileged"},
				Resources:     []string{"securitycontextconstraints"},
				Verbs:         []string{"use"},
			},
			{
				APIGroups:     []string{"policy"},
				ResourceNames: []string{c.K8sSensorResourcesName()},
				Resources:     []string{"podsecuritypolicies"},
				Verbs:         []string{"use"},
			},
		},
	}
}

func (c *clusterRoleBuilder) Build() optional.Optional[client.Object] {
	return optional.Of(c.build())
}

func NewClusterRoleBuilder(agent *instanav1.InstanaAgent, isOpenShift bool) builder.ObjectBuilder {
	return newRbacBuilder(
		agent, isOpenShift, &clusterRoleBuilder{
			InstanaAgent: agent,
			Helpers:      helpers.NewHelpers(agent),
		},
	)
}
