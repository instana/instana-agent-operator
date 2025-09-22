/*
(c) Copyright IBM Corp. 2024, 2025

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

type roleBindingBuilder struct {
	*instanav1.InstanaAgent
	helpers.Helpers
}

func (rb *roleBindingBuilder) ComponentName() string {
	return constants.ComponentK8Sensor
}

func (rb *roleBindingBuilder) IsNamespaced() bool {
	return true
}

func (rb *roleBindingBuilder) Build() optional.Optional[client.Object] {
	return optional.Of[client.Object](
		&rbacv1.RoleBinding{
			TypeMeta: metav1.TypeMeta{
				APIVersion: rbacApiVersion,
				Kind:       "RoleBinding",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      rb.K8sSensorResourcesName() + "-etcd-reader",
				Namespace: "kube-system",
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      rb.K8sSensorResourcesName(),
					Namespace: rb.Namespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     rb.K8sSensorResourcesName() + "-etcd-reader",
			},
		},
	)
}

func NewRoleBindingBuilder(agent *instanav1.InstanaAgent) builder.ObjectBuilder {
	return &roleBindingBuilder{
		InstanaAgent: agent,
		Helpers:      helpers.NewHelpers(agent),
	}
}