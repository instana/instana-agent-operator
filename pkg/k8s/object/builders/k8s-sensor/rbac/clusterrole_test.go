package rbac

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

func TestClusterRoleBuilder_IsNamespaced_ComponentName(t *testing.T) {
	assertions := require.New(t)

	cb := NewClusterRoleBuilder(nil)

	assertions.False(cb.IsNamespaced())
	assertions.Equal(constants.ComponentK8Sensor, cb.ComponentName())
}

func TestClusterRoleBuilder_Build(t *testing.T) {
	assertions := require.New(t)
	ctrl := gomock.NewController(t)

	sensorResourcesName := rand.String(10)

	expected := optional.Of[client.Object](
		&rbacv1.ClusterRole{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       "ClusterRole",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: sensorResourcesName,
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
					Verbs:     []string{"get", "list", "watch"},
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
					Verbs: []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"apps"},
					Resources: []string{"daemonsets", "deployments", "replicasets", "statefulsets"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"batch"},
					Resources: []string{"cronjobs", "jobs"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"networking.k8s.io"},
					Resources: []string{"ingresses"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"autoscaling"},
					Resources: []string{"horizontalpodautoscalers"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"apps.openshift.io"},
					Resources: []string{"deploymentconfigs"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups:     []string{"security.openshift.io"},
					ResourceNames: []string{"privileged"},
					Resources:     []string{"securitycontextconstraints"},
					Verbs:         []string{"use"},
				},
				{
					APIGroups:     []string{"policy"},
					ResourceNames: []string{sensorResourcesName},
					Resources:     []string{"podsecuritypolicies"},
					Verbs:         []string{"use"},
				},
			},
		},
	)

	helpers := NewMockHelpers(ctrl)
	helpers.EXPECT().K8sSensorResourcesName().Return(sensorResourcesName).Times(2)

	cb := &clusterRoleBuilder{
		Helpers: helpers,
	}

	actual := cb.Build()

	assertions.Equal(expected, actual)
}
