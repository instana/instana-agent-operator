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

package keys_secret

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	backends "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/backends"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

type secretBuilder struct {
	*instanav1.InstanaAgentRemote
	additionalBackends []backends.RemoteSensorBackend
}

func NewSecretBuilder(agent *instanav1.InstanaAgentRemote, backends []backends.RemoteSensorBackend) builder.ObjectBuilder {
	return &secretBuilder{
		InstanaAgentRemote: agent,
		additionalBackends: backends,
	}
}

func (s *secretBuilder) IsNamespaced() bool {
	return true
}

func (s *secretBuilder) ComponentName() string {
	return constants.ComponentInstanaAgentRemote
}

func (s *secretBuilder) Build() optional.Optional[client.Object] {
	switch s.Spec.Agent.KeysSecret {
	case "":
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
			Name:      s.Name,
			Namespace: s.Namespace,
		},
		Data: s.getData(),
		Type: corev1.SecretTypeOpaque,
	}
}

func (s *secretBuilder) getData() map[string][]byte {
	useSecretMounts := s.Spec.UseSecretMounts == nil || *s.Spec.UseSecretMounts
	data := make(map[string][]byte, len(s.additionalBackends)+8)

	optional.Of(s.Spec.Agent.DownloadKey).IfPresent(
		func(downloadKey string) {
			data[constants.DownloadKey] = []byte(downloadKey)
			if useSecretMounts {
				data[constants.SecretFileDownloadKey] = []byte(downloadKey)
			}
		},
	)

	if useSecretMounts {
		optional.Of(s.Spec.Agent.ProxyUser).IfPresent(
			func(proxyUser string) {
				data[constants.SecretKeyProxyUser] = []byte(proxyUser)
			},
		)

		optional.Of(s.Spec.Agent.ProxyPassword).IfPresent(
			func(proxyPassword string) {
				data[constants.SecretKeyProxyPassword] = []byte(proxyPassword)
			},
		)

		optional.Of(s.Spec.Agent.MirrorReleaseRepoUsername).IfPresent(
			func(username string) {
				data[constants.SecretKeyMirrorReleaseRepoUsername] = []byte(username)
			},
		)

		optional.Of(s.Spec.Agent.MirrorReleaseRepoPassword).IfPresent(
			func(password string) {
				data[constants.SecretKeyMirrorReleaseRepoPassword] = []byte(password)
			},
		)

		optional.Of(s.Spec.Agent.MirrorSharedRepoUsername).IfPresent(
			func(username string) {
				data[constants.SecretKeyMirrorSharedRepoUsername] = []byte(username)
			},
		)

		optional.Of(s.Spec.Agent.MirrorSharedRepoPassword).IfPresent(
			func(password string) {
				data[constants.SecretKeyMirrorSharedRepoPassword] = []byte(password)
			},
		)

		if proxyValue := s.httpsProxyValue(); proxyValue != "" {
			data[constants.SecretKeyHttpsProxy] = []byte(proxyValue)
		}
	}

	for _, backend := range s.additionalBackends {
		optional.Of(backend.EndpointKey).IfPresent(
			func(key string) {
				data[constants.AgentKey+backend.ResourceSuffix] = []byte(key)
				if useSecretMounts && backend.ResourceSuffix == "" {
					data[constants.SecretFileAgentKey] = []byte(key)
				}
			},
		)
	}

	return data
}

func (s *secretBuilder) httpsProxyValue() string {
	if s.Spec.Agent.ProxyHost == "" {
		return ""
	}

	protocol := optional.Of(s.Spec.Agent.ProxyProtocol).GetOrDefault("http")
	port := optional.Of(s.Spec.Agent.ProxyPort).GetOrDefault("80")

	if s.Spec.Agent.ProxyUser == "" || s.Spec.Agent.ProxyPassword == "" {
		return fmt.Sprintf("%s://%s:%s", protocol, s.Spec.Agent.ProxyHost, port)
	}

	return fmt.Sprintf(
		"%s://%s%s:%s",
		protocol,
		s.Spec.Agent.ProxyUser+":"+s.Spec.Agent.ProxyPassword+"@",
		s.Spec.Agent.ProxyHost,
		port,
	)
}
