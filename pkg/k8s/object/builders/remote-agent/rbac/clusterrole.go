/*
(c) Copyright IBM Corp. 2025
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

	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
)

type clusterRoleBuilder struct {
	*instanav1.RemoteAgent
	helpers.RemoteHelpers
}

func (c *clusterRoleBuilder) ComponentName() string {
	return constants.ComponentRemoteAgent
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
				Name: c.ServiceAccountName(),
			},
			Rules: []rbacv1.PolicyRule{
				{
					// Needed for Prometheus / Kubelet metrics access
					NonResourceURLs: []string{
						"/metrics",
						"/stats/summary",
						"/metrics/cadvisor",
					},
					Verbs: []string{"get"},
				},
				{
					// Allow read-only access to pods and nodes (metrics)
					APIGroups: []string{""},
					Resources: []string{
						"pods",
						"nodes",
					},
					Verbs: constants.ReaderVerbs(), // typically ["get", "list", "watch"]
				},
			},
		},
	)
}

func NewClusterRoleBuilder(agent *instanav1.RemoteAgent) builder.ObjectBuilder {
	return &clusterRoleBuilder{
		RemoteAgent:   agent,
		RemoteHelpers: helpers.NewRemoteHelpers(agent),
	}
}
