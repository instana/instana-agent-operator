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

type clusterRoleBindingBuilder struct {
	*instanav1.InstanaAgent
	helpers.Helpers
}

func (c *clusterRoleBindingBuilder) IsNamespaced() bool {
	return false
}

func (c *clusterRoleBindingBuilder) ComponentName() string {
	return constants.ComponentInstanaAgent
}

func (c *clusterRoleBindingBuilder) Build() optional.Optional[client.Object] {
	return optional.Of[client.Object](
		&rbacv1.ClusterRoleBinding{
			TypeMeta: metav1.TypeMeta{
				APIVersion: rbacApiVersion,
				Kind:       "ClusterRoleBinding",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: c.ServiceAccountName(),
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacApiGroup,
				Kind:     roleKind,
				Name:     c.ServiceAccountName(),
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      subjectKind,
					Name:      c.ServiceAccountName(),
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
