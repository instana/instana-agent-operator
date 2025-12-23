/*
(c) Copyright IBM Corp. 2024
(c) Copyright Instana Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
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

func (c *clusterRoleBuilder) Build() optional.Optional[client.Object] {
	return optional.Of[client.Object](
		&rbacv1.ClusterRole{
			TypeMeta: metav1.TypeMeta{
				APIVersion: rbacApiVersion,
				Kind:       roleKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: c.K8sSensorResourcesName(),
			},
			Rules: []rbacv1.PolicyRule{
				{
					NonResourceURLs: []string{"/version", "/healthz", "/metrics", "/metrics/*"},
					Verbs:           []string{"get"},
					APIGroups:       []string{},
					Resources:       []string{},
				},
				{
					APIGroups: []string{"apiextensions.k8s.io"},
					Resources: []string{"customresourcedefinitions"},
					Verbs:     constants.ReaderVerbs(),
				},
				{
					APIGroups: []string{"extensions"},
					Resources: []string{"deployments", "replicasets", "ingresses"},
					Verbs:     constants.ReaderVerbs(),
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
						"nodes/metrics",
						"nodes/stats",
						"nodes/proxy",
						"pods",
						"pods/log",
						"replicationcontrollers",
						"resourcequotas",
						"persistentvolumes",
						"persistentvolumeclaims",
					},
					Verbs: constants.ReaderVerbs(),
				},
				{
					APIGroups: []string{"apps"},
					Resources: []string{"daemonsets", "deployments", "replicasets", "statefulsets"},
					Verbs:     constants.ReaderVerbs(),
				},
				{
					APIGroups: []string{"batch"},
					Resources: []string{"cronjobs", "jobs"},
					Verbs:     constants.ReaderVerbs(),
				},
				{
					APIGroups: []string{"networking.k8s.io"},
					Resources: []string{"ingresses"},
					Verbs:     constants.ReaderVerbs(),
				},
				{
					APIGroups: []string{"autoscaling"},
					Resources: []string{"horizontalpodautoscalers"},
					Verbs:     constants.ReaderVerbs(),
				},
				{
					APIGroups: []string{"apps.openshift.io"},
					Resources: []string{"deploymentconfigs"},
					Verbs:     constants.ReaderVerbs(),
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
		},
	)
}

func NewClusterRoleBuilder(agent *instanav1.InstanaAgent) builder.ObjectBuilder {
	return &clusterRoleBuilder{
		InstanaAgent: agent,
		Helpers:      helpers.NewHelpers(agent),
	}
}
