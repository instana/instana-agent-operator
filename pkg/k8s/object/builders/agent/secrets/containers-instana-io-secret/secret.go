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

package containers_instana_io_secret

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/json_or_die"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

type DockerConfigAuth struct {
	Auth []byte `json:"auth"`
}

type DockerConfigJson struct {
	Auths map[string]DockerConfigAuth `json:"auths"`
}

type secretBuilder struct {
	instanaAgent *instanav1.InstanaAgent
	helpers      helpers.Helpers
	marshaler    json_or_die.JsonOrDieMarshaler[*DockerConfigJson]
}

func NewSecretBuilder(agent *instanav1.InstanaAgent) builder.ObjectBuilder {
	return &secretBuilder{
		instanaAgent: agent,
		helpers:      helpers.NewHelpers(agent),
		marshaler:    json_or_die.NewJsonOrDie[DockerConfigJson](),
	}
}

func (s *secretBuilder) Build() optional.Optional[client.Object] {
	switch s.helpers.UseContainersSecret() {
	case true:
		return optional.Of[client.Object](s.build())
	default:
		return optional.Empty[client.Object]()
	}
}

func (s *secretBuilder) IsNamespaced() bool {
	return true
}

func (s *secretBuilder) ComponentName() string {
	return constants.ComponentInstanaAgent
}

func (s *secretBuilder) buildDockerConfigJson() []byte {
	password := optional.Of(s.instanaAgent.Spec.Agent.DownloadKey).GetOrDefault(s.instanaAgent.Spec.Agent.Key)
	auth := fmt.Sprintf("_:%s", password)

	json := DockerConfigJson{
		Auths: map[string]DockerConfigAuth{
			helpers.ContainersInstanaIORegistry: {
				Auth: []byte(auth),
			},
		},
	}

	return s.marshaler.MarshalOrDie(&json)
}

func (s *secretBuilder) build() *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.helpers.ContainersSecretName(),
			Namespace: s.instanaAgent.Namespace,
		},
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: s.buildDockerConfigJson(),
		},
		Type: corev1.SecretTypeDockerConfigJson,
	}
}
