/*
(c) Copyright IBM Corp. 2025

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
package serviceaccount

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

type serviceAccountBuilder struct {
	*instanav1.RemoteAgent
	helpers.RemoteHelpers
}

func (s *serviceAccountBuilder) IsNamespaced() bool {
	return true
}

func (s *serviceAccountBuilder) ComponentName() string {
	return constants.ComponentRemoteAgent
}

func (s *serviceAccountBuilder) build() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "remote-agent",
			Namespace:   s.Namespace,
			Annotations: s.Spec.ServiceAccountSpec.Annotations,
		},
	}
}

func (s *serviceAccountBuilder) Build() optional.Optional[client.Object] {
	if pointer.DerefOrEmpty(s.Spec.ServiceAccountSpec.Create.Create) {
		return optional.Of[client.Object](s.build())
	} else {
		return optional.Empty[client.Object]()
	}
}

func NewServiceAccountBuilder(agent *instanav1.RemoteAgent) builder.ObjectBuilder {
	return &serviceAccountBuilder{
		RemoteAgent:   agent,
		RemoteHelpers: helpers.NewRemoteHelpers(agent),
	}
}
