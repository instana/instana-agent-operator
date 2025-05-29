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

// container package is a builder responsible of generating a k8s Secret-resource for containers.instana.io
// with a download key specified either in keysSecret or in the Agent.Spec as a downloadKey or just key
package secrets

import (
	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/json_or_die"
	commonbuilder "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/optional"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type dockerConfigAuth struct {
	Auth []byte `json:"auth"`
}

type dockerConfig struct {
	Auths map[string]dockerConfigAuth `json:"auths"`
}

// NewBuilder creates a builder struct implementing common builders interface
func NewContainerBuilder(agent *instanav1.RemoteAgent, keysSecret *corev1.Secret) commonbuilder.ObjectBuilder {
	return &containerBuilder{
		remoteAgent: agent,
		keysSecret:  keysSecret,
		helpers:     helpers.NewRemoteHelpers(agent),
		marshaler:   json_or_die.NewJsonOrDie[dockerConfig](),
	}
}

type containerBuilder struct {
	remoteAgent *instanav1.RemoteAgent
	helpers     helpers.RemoteHelpers
	marshaler   json_or_die.JsonOrDieMarshaler[*dockerConfig]
	keysSecret  *corev1.Secret
}

func (s *containerBuilder) IsNamespaced() bool {
	return true
}

func (s *containerBuilder) ComponentName() string {
	return constants.ComponentRemoteAgent
}

// Build generates a v1.Secret if Agent.ImageSpec.Name contains "containers.instana.io" and the key for the secret is found
func (s *containerBuilder) Build() optional.Optional[client.Object] {
	switch s.helpers.UseContainersSecret() {
	case true:
		return optional.Of[client.Object](s.build())
	default:
		return optional.Empty[client.Object]()
	}
}

// build creates the boilerplate v1.Secret and if key was found and marshals DockerConfig to the Data property
func (s *containerBuilder) build() *corev1.Secret {
	if downloadKey := s.downloadKey(); downloadKey != nil {
		return &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      s.helpers.ContainersSecretName(),
				Namespace: s.remoteAgent.Namespace,
			},
			Data: map[string][]byte{
				corev1.DockerConfigJsonKey: s.marshalJSON(*downloadKey),
			},
			Type: corev1.SecretTypeDockerConfigJson,
		}
	}

	return nil
}

// downloadKey fetches download key firstly by attempting to find it in the keysSecret field and then from Agent config
func (s *containerBuilder) downloadKey() *string {
	if s.keysSecret != nil {
		if downloadKeyValueFromSecret, ok := s.keysSecret.Data["downloadKey"]; ok {
			str := string(downloadKeyValueFromSecret)
			return &str
		} else if keyValueFromSecret, ok := s.keysSecret.Data["key"]; ok {
			str := string(keyValueFromSecret)
			return &str
		}
	}

	if s.remoteAgent.Spec.Agent.DownloadKey != "" {
		return &s.remoteAgent.Spec.Agent.DownloadKey
	} else if s.remoteAgent.Spec.Agent.Key != "" {
		return &s.remoteAgent.Spec.Agent.Key
	}

	return nil
}

// marshalJSON is responsible for making a []byte out of DockerConfig or panic
func (s *containerBuilder) marshalJSON(downloadKey string) []byte {
	return s.marshaler.MarshalOrDie(
		&dockerConfig{
			Auths: map[string]dockerConfigAuth{
				helpers.ContainersInstanaIORegistry: {
					Auth: []byte("_:" + downloadKey),
				},
			},
		},
	)
}
