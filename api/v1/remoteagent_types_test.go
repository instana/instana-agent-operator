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
package v1

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/instana/instana-agent-operator/pkg/pointer"
)

func TestRemoteAgent_Default(t *testing.T) {
	defaultTrue := pointer.To(true)

	mainAgent := InstanaAgent{
		Spec: InstanaAgentSpec{
			Agent: BaseAgentSpec{
				EndpointHost: "custom-host.instana.io",
				EndpointPort: "8443",
				ExtendedImageSpec: ExtendedImageSpec{
					ImageSpec: ImageSpec{
						Name:       "custom/agent",
						Tag:        "1.2.3",
						PullPolicy: corev1.PullIfNotPresent,
					},
				},
				Key:               "agent-key",
				DownloadKey:       "download-key",
				KeysSecret:        "instana-keys",
				ListenAddress:     "0.0.0.0",
				MinReadySeconds:   10,
				ConfigurationYaml: "config-yaml",
				AdditionalBackends: []BackendSpec{
					{
						EndpointHost: "backend1",
						EndpointPort: "443",
						Key:          "test",
					},
					{
						EndpointHost: "backend2",
						EndpointPort: "443",
						Key:          "test",
					},
				},
				TlsSpec:                   TlsSpec{SecretName: "tls-secret"},
				ProxyHost:                 "proxy-host",
				ProxyPort:                 "3128",
				ProxyProtocol:             "http",
				ProxyUseDNS:               true,
				ProxyUser:                 "proxy-user",
				ProxyPassword:             "proxy-pass",
				Env:                       map[string]string{"ENV_VAR": "value"},
				RedactKubernetesSecrets:   "true",
				MvnRepoFeaturesPath:       "/mvn/features",
				MvnRepoSharedPath:         "/mvn/shared",
				MvnRepoUrl:                "http://mvn.repo",
				MirrorReleaseRepoUsername: "release-user",
				MirrorReleaseRepoPassword: "release-pass",
				MirrorReleaseRepoUrl:      "http://release.repo",
				MirrorSharedRepoUsername:  "shared-user",
				MirrorSharedRepoPassword:  "shared-pass",
				MirrorSharedRepoUrl:       "http://shared.repo",
			},
			Cluster: Name{Name: "test-cluster"},
			Rbac:    Create{Create: defaultTrue},
			ServiceAccountSpec: ServiceAccountSpec{
				Create: Create{Create: defaultTrue},
			},
		},
	}

	tests := []struct {
		name     string
		spec     *RemoteAgentSpec
		expected *RemoteAgentSpec
	}{
		{
			name: "defaults_from_main_agent",
			spec: &RemoteAgentSpec{
				ConfigurationYaml: "config-yaml",
			},
			expected: &RemoteAgentSpec{
				ConfigurationYaml: "config-yaml",
				Agent: BaseAgentSpec{
					EndpointHost: "custom-host.instana.io",
					EndpointPort: "8443",
					ExtendedImageSpec: ExtendedImageSpec{
						ImageSpec: ImageSpec{
							Name:       "custom/agent",
							Tag:        "1.2.3",
							PullPolicy: corev1.PullIfNotPresent,
						},
					},
					Key:               "agent-key",
					DownloadKey:       "download-key",
					KeysSecret:        "instana-keys",
					ListenAddress:     "0.0.0.0",
					MinReadySeconds:   10,
					ConfigurationYaml: "config-yaml",
					AdditionalBackends: []BackendSpec{
						{
							EndpointHost: "backend1",
							EndpointPort: "443",
							Key:          "test",
						},
						{
							EndpointHost: "backend2",
							EndpointPort: "443",
							Key:          "test",
						},
					},
					TlsSpec:                   TlsSpec{SecretName: "tls-secret"},
					ProxyHost:                 "proxy-host",
					ProxyPort:                 "3128",
					ProxyProtocol:             "http",
					ProxyUseDNS:               true,
					ProxyUser:                 "proxy-user",
					ProxyPassword:             "proxy-pass",
					RedactKubernetesSecrets:   "true",
					MvnRepoFeaturesPath:       "/mvn/features",
					MvnRepoSharedPath:         "/mvn/shared",
					MvnRepoUrl:                "http://mvn.repo",
					MirrorReleaseRepoUsername: "release-user",
					MirrorReleaseRepoPassword: "release-pass",
					MirrorReleaseRepoUrl:      "http://release.repo",
					MirrorSharedRepoUsername:  "shared-user",
					MirrorSharedRepoPassword:  "shared-pass",
					MirrorSharedRepoUrl:       "http://shared.repo",
				},
				Cluster: Name{
					Name: "test-cluster",
				},
				Rbac: Create{Create: defaultTrue},
				ServiceAccountSpec: ServiceAccountSpec{
					Create: Create{Create: defaultTrue},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertions := require.New(t)

			ra := &RemoteAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-remote-agent",
					Namespace: "default",
				},
				Spec: *tt.spec,
			}

			ra.DefaultWithHost(mainAgent)

			assertions.Equal(tt.expected, &ra.Spec)
		})
	}
}
