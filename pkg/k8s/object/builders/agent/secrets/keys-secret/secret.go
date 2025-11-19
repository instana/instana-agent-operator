/*
(c) Copyright IBM Corp. 2024, 2025
*/

package keys_secret

import (
	"fmt"
	"net/url"

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
	*instanav1.InstanaAgent
	backends []backends.K8SensorBackend
}

func NewSecretBuilder(
	agent *instanav1.InstanaAgent,
	backends []backends.K8SensorBackend,
) builder.ObjectBuilder {
	return &secretBuilder{
		InstanaAgent: agent,
		backends:     backends,
	}
}

func (s *secretBuilder) IsNamespaced() bool {
	return true
}

func (s *secretBuilder) ComponentName() string {
	return constants.ComponentInstanaAgent
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
	data := make(map[string][]byte, len(s.backends)+8) // Increased capacity for all secrets

	// Agent keys
	optional.Of(s.Spec.Agent.DownloadKey).IfPresent(
		func(downloadKey string) {
			data[constants.DownloadKey] = []byte(downloadKey)
			// Only add environment variable name for file mounting if UseSecretMounts is true
			if s.Spec.UseSecretMounts != nil && *s.Spec.UseSecretMounts {
				data[constants.SecretFileDownloadKey] = []byte(downloadKey)
			}
		},
	)

	// Proxy credentials
	optional.Of(s.Spec.Agent.ProxyUser).IfPresent(
		func(proxyUser string) {
			if s.Spec.UseSecretMounts != nil && *s.Spec.UseSecretMounts {
				data[constants.SecretFileProxyUser] = []byte(proxyUser)
			}
		},
	)

	optional.Of(s.Spec.Agent.ProxyPassword).IfPresent(
		func(proxyPassword string) {
			if s.Spec.UseSecretMounts != nil && *s.Spec.UseSecretMounts {
				data[constants.SecretFileProxyPassword] = []byte(proxyPassword)
			}
		},
	)

	// Mirror repository credentials
	optional.Of(s.Spec.Agent.MirrorReleaseRepoUsername).IfPresent(
		func(username string) {
			if s.Spec.UseSecretMounts != nil && *s.Spec.UseSecretMounts {
				data[constants.SecretFileMirrorReleaseRepoUsername] = []byte(username)
			}
		},
	)

	optional.Of(s.Spec.Agent.MirrorReleaseRepoPassword).IfPresent(
		func(password string) {
			if s.Spec.UseSecretMounts != nil && *s.Spec.UseSecretMounts {
				data[constants.SecretFileMirrorReleaseRepoPassword] = []byte(password)
			}
		},
	)

	optional.Of(s.Spec.Agent.MirrorSharedRepoUsername).IfPresent(
		func(username string) {
			if s.Spec.UseSecretMounts != nil && *s.Spec.UseSecretMounts {
				data[constants.SecretFileMirrorSharedRepoUsername] = []byte(username)
			}
		},
	)

	optional.Of(s.Spec.Agent.MirrorSharedRepoPassword).IfPresent(
		func(password string) {
			if s.Spec.UseSecretMounts != nil && *s.Spec.UseSecretMounts {
				data[constants.SecretFileMirrorSharedRepoPassword] = []byte(password)
			}
		},
	)

	// HTTPS_PROXY
	if s.Spec.Agent.ProxyHost != "" {
		// Only add if proxy host is set
		if s.Spec.UseSecretMounts != nil && *s.Spec.UseSecretMounts {
			// Generate the HTTPS_PROXY value similar to how it's done in env_builder.go
			var proxyValue string
			u := url.URL{
				Scheme: optional.Of(s.Spec.Agent.ProxyProtocol).GetOrDefault("http"),
				Host: fmt.Sprintf(
					"%s:%s",
					s.Spec.Agent.ProxyHost,
					optional.Of(s.Spec.Agent.ProxyPort).GetOrDefault("80"),
				),
			}
			if s.Spec.Agent.ProxyUser != "" && s.Spec.Agent.ProxyPassword != "" {
				u.User = url.UserPassword(s.Spec.Agent.ProxyUser, s.Spec.Agent.ProxyPassword)
			}
			proxyValue = u.String()
			data[constants.SecretFileHttpsProxy] = []byte(proxyValue)
		}
	}

	// Backend keys
	for _, backend := range s.backends {
		optional.Of(backend.EndpointKey).IfPresent(
			func(key string) {
				data[constants.AgentKey+backend.ResourceSuffix] = []byte(key)
				// For all backends, also add with environment variable name for file mounting
				// This ensures each k8sensor deployment can read the key from the mounted secret file
				if s.Spec.UseSecretMounts != nil && *s.Spec.UseSecretMounts {
					data[constants.SecretFileAgentKey+backend.ResourceSuffix] = []byte(key)
				}
			},
		)
	}

	return data
}
