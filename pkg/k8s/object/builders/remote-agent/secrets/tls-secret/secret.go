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

package tls_secret

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

type secretBuilder struct {
	*instanav1.RemoteAgent

	helpers.RemoteHelpers
}

func NewSecretBuilder(agent *instanav1.RemoteAgent) builder.ObjectBuilder {
	return &secretBuilder{
		RemoteAgent: agent,

		RemoteHelpers: helpers.NewRemoteHelpers(agent),
	}
}

func (s *secretBuilder) IsNamespaced() bool {
	return true
}

func (s *secretBuilder) ComponentName() string {
	return constants.ComponentRemoteAgent
}

func (s *secretBuilder) Build() optional.Optional[client.Object] {
	switch tls := s.Spec.Agent.TlsSpec; tls.SecretName == "" && len(tls.Key) > 0 && len(tls.Certificate) > 0 {
	case true:
		return optional.Of[client.Object](s.build())
	default:
		return optional.Empty[client.Object]()
	}
}

func (s *secretBuilder) build() *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.TLSSecretName(),
			Namespace: s.Namespace,
		},
		Data: map[string][]byte{
			corev1.TLSCertKey:       s.Spec.Agent.TlsSpec.Certificate,
			corev1.TLSPrivateKeyKey: s.Spec.Agent.TlsSpec.Key,
		},
		Type: corev1.SecretTypeTLS,
	}
}
