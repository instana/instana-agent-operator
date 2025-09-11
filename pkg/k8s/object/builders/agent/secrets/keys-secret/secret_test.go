/*
(c) Copyright IBM Corp. 2024, 2025
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
package keys_secret

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	backends "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/backends"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

func TestSecretBuilder_IsNamespaced_ComponentName(t *testing.T) {
	assertions := require.New(t)

	s := NewSecretBuilder(&instanav1.InstanaAgent{}, make([]backends.K8SensorBackend, 0))

	assertions.True(s.IsNamespaced())
	assertions.Equal("instana-agent", s.ComponentName())
}

func randString() string {
	return rand.String(rand.IntnRange(1, 15))
}

func emptyOrRandomString() []string {
	return []string{"", randString()}
}

func TestSecretBuilder_Build(t *testing.T) {
	for _, keysSecret := range emptyOrRandomString() {
		for _, key := range emptyOrRandomString() {
			for _, downloadKey := range emptyOrRandomString() {
				t.Run(
					fmt.Sprintf(
						"keysSecretIsEmpty:%v_keyIsEmpty:%v_downloadKeyIsEmpty:%v",
						len(keysSecret) == 0,
						len(key) == 0,
						len(downloadKey) == 0,
					), func(t *testing.T) {
						assertions := require.New(t)

						name := randString()
						namespace := randString()

						agent := instanav1.InstanaAgent{
							ObjectMeta: metav1.ObjectMeta{
								Name:      name,
								Namespace: namespace,
							},
							Spec: instanav1.InstanaAgentSpec{
								Agent: instanav1.BaseAgentSpec{
									KeysSecret:  keysSecret,
									Key:         key,
									DownloadKey: downloadKey,
								},
							},
						}

						backend := backends.NewK8SensorBackend("", key, downloadKey, "", "")
						var backends [1]backends.K8SensorBackend
						backends[0] = *backend

						sb := NewSecretBuilder(&agent, backends[:])

						actual := sb.Build()

						switch keysSecret {
						case "":
							data := make(map[string][]byte, 2)

							if len(key) > 0 {
								data["key"] = []byte(key)
							}

							if len(downloadKey) > 0 {
								data["downloadKey"] = []byte(downloadKey)
							}

							expected := optional.Of[client.Object](
								&corev1.Secret{
									TypeMeta: metav1.TypeMeta{
										APIVersion: "v1",
										Kind:       "Secret",
									},
									ObjectMeta: metav1.ObjectMeta{
										Name:      name,
										Namespace: namespace,
									},
									Data: data,
									Type: corev1.SecretTypeOpaque,
								},
							)

							assertions.Equal(expected, actual)
						default:
							assertions.Empty(actual)
						}
					},
				)
			}
		}
	}
}

func TestSecretBuilder_BuildWithSecretMounts(t *testing.T) {
	testCases := []struct {
		name           string
		agent          *instanav1.InstanaAgent
		backends       []backends.K8SensorBackend
		expectedSecret *corev1.Secret
	}{
		{
			name: "with UseSecretMounts true and agent key and download key",
			agent: &instanav1.InstanaAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "test-namespace",
				},
				Spec: instanav1.InstanaAgentSpec{
					UseSecretMounts: pointer.To(true),
					Agent: instanav1.BaseAgentSpec{
						Key:         "test-key",
						DownloadKey: "test-download-key",
					},
				},
			},
			backends: []backends.K8SensorBackend{
				{
					ResourceSuffix: "",
					EndpointKey:    "test-key",
				},
			},
			expectedSecret: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"key":                  []byte("test-key"),
					"downloadKey":          []byte("test-download-key"),
					"INSTANA_AGENT_KEY":    []byte("test-key"),
					"INSTANA_DOWNLOAD_KEY": []byte("test-download-key"),
				},
				Type: corev1.SecretTypeOpaque,
			},
		},
		{
			name: "with UseSecretMounts true and proxy credentials",
			agent: &instanav1.InstanaAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "test-namespace",
				},
				Spec: instanav1.InstanaAgentSpec{
					UseSecretMounts: pointer.To(true),
					Agent: instanav1.BaseAgentSpec{
						Key:           "test-key",
						DownloadKey:   "test-download-key",
						ProxyUser:     "proxy-user",
						ProxyPassword: "proxy-password",
					},
				},
			},
			backends: []backends.K8SensorBackend{
				{
					ResourceSuffix: "",
					EndpointKey:    "test-key",
				},
			},
			expectedSecret: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"key":                          []byte("test-key"),
					"downloadKey":                  []byte("test-download-key"),
					"INSTANA_AGENT_KEY":            []byte("test-key"),
					"INSTANA_DOWNLOAD_KEY":         []byte("test-download-key"),
					"INSTANA_AGENT_PROXY_USER":     []byte("proxy-user"),
					"INSTANA_AGENT_PROXY_PASSWORD": []byte("proxy-password"),
				},
				Type: corev1.SecretTypeOpaque,
			},
		},
		{
			name: "with UseSecretMounts true and mirror repository credentials",
			agent: &instanav1.InstanaAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "test-namespace",
				},
				Spec: instanav1.InstanaAgentSpec{
					UseSecretMounts: pointer.To(true),
					Agent: instanav1.BaseAgentSpec{
						Key:                       "test-key",
						DownloadKey:               "test-download-key",
						MirrorReleaseRepoUsername: "release-user",
						MirrorReleaseRepoPassword: "release-password",
						MirrorSharedRepoUsername:  "shared-user",
						MirrorSharedRepoPassword:  "shared-password",
					},
				},
			},
			backends: []backends.K8SensorBackend{
				{
					ResourceSuffix: "",
					EndpointKey:    "test-key",
				},
			},
			expectedSecret: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"key":                  []byte("test-key"),
					"downloadKey":          []byte("test-download-key"),
					"INSTANA_AGENT_KEY":    []byte("test-key"),
					"INSTANA_DOWNLOAD_KEY": []byte("test-download-key"),
					"AGENT_RELEASE_REPOSITORY_MIRROR_USERNAME":  []byte("release-user"),
					"AGENT_RELEASE_REPOSITORY_MIRROR_PASSWORD":  []byte("release-password"),
					"INSTANA_SHARED_REPOSITORY_MIRROR_USERNAME": []byte("shared-user"),
					"INSTANA_SHARED_REPOSITORY_MIRROR_PASSWORD": []byte("shared-password"),
				},
				Type: corev1.SecretTypeOpaque,
			},
		},
		{
			name: "with UseSecretMounts true and HTTPS_PROXY with host only",
			agent: &instanav1.InstanaAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "test-namespace",
				},
				Spec: instanav1.InstanaAgentSpec{
					UseSecretMounts: pointer.To(true),
					Agent: instanav1.BaseAgentSpec{
						Key:         "test-key",
						DownloadKey: "test-download-key",
						ProxyHost:   "proxy.example.com",
					},
				},
			},
			backends: []backends.K8SensorBackend{
				{
					ResourceSuffix: "",
					EndpointKey:    "test-key",
				},
			},
			expectedSecret: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"key":                  []byte("test-key"),
					"downloadKey":          []byte("test-download-key"),
					"INSTANA_AGENT_KEY":    []byte("test-key"),
					"INSTANA_DOWNLOAD_KEY": []byte("test-download-key"),
					"HTTPS_PROXY":          []byte("http://proxy.example.com:80"),
				},
				Type: corev1.SecretTypeOpaque,
			},
		},
		{
			name: "with UseSecretMounts true and HTTPS_PROXY with host, port, protocol",
			agent: &instanav1.InstanaAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "test-namespace",
				},
				Spec: instanav1.InstanaAgentSpec{
					UseSecretMounts: pointer.To(true),
					Agent: instanav1.BaseAgentSpec{
						Key:           "test-key",
						DownloadKey:   "test-download-key",
						ProxyHost:     "proxy.example.com",
						ProxyPort:     "8080",
						ProxyProtocol: "https",
					},
				},
			},
			backends: []backends.K8SensorBackend{
				{
					ResourceSuffix: "",
					EndpointKey:    "test-key",
				},
			},
			expectedSecret: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"key":                  []byte("test-key"),
					"downloadKey":          []byte("test-download-key"),
					"INSTANA_AGENT_KEY":    []byte("test-key"),
					"INSTANA_DOWNLOAD_KEY": []byte("test-download-key"),
					"HTTPS_PROXY":          []byte("https://proxy.example.com:8080"),
				},
				Type: corev1.SecretTypeOpaque,
			},
		},
		{
			name: "with UseSecretMounts true and HTTPS_PROXY with credentials",
			agent: &instanav1.InstanaAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "test-namespace",
				},
				Spec: instanav1.InstanaAgentSpec{
					UseSecretMounts: pointer.To(true),
					Agent: instanav1.BaseAgentSpec{
						Key:           "test-key",
						DownloadKey:   "test-download-key",
						ProxyHost:     "proxy.example.com",
						ProxyPort:     "8080",
						ProxyProtocol: "https",
						ProxyUser:     "proxy-user",
						ProxyPassword: "proxy-password",
					},
				},
			},
			backends: []backends.K8SensorBackend{
				{
					ResourceSuffix: "",
					EndpointKey:    "test-key",
				},
			},
			expectedSecret: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"key":                          []byte("test-key"),
					"downloadKey":                  []byte("test-download-key"),
					"INSTANA_AGENT_KEY":            []byte("test-key"),
					"INSTANA_DOWNLOAD_KEY":         []byte("test-download-key"),
					"INSTANA_AGENT_PROXY_USER":     []byte("proxy-user"),
					"INSTANA_AGENT_PROXY_PASSWORD": []byte("proxy-password"),
					"HTTPS_PROXY": []byte(
						"https://proxy-user:proxy-password@proxy.example.com:8080",
					),
				},
				Type: corev1.SecretTypeOpaque,
			},
		},
		{
			name: "with UseSecretMounts true and multiple backends",
			agent: &instanav1.InstanaAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "test-namespace",
				},
				Spec: instanav1.InstanaAgentSpec{
					UseSecretMounts: pointer.To(true),
					Agent: instanav1.BaseAgentSpec{
						Key:         "test-key",
						DownloadKey: "test-download-key",
					},
				},
			},
			backends: []backends.K8SensorBackend{
				{
					ResourceSuffix: "",
					EndpointKey:    "first-backend-key",
				},
				{
					ResourceSuffix: "-2",
					EndpointKey:    "second-backend-key",
				},
			},
			expectedSecret: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"key":                  []byte("first-backend-key"),
					"key-2":                []byte("second-backend-key"),
					"downloadKey":          []byte("test-download-key"),
					"INSTANA_AGENT_KEY":    []byte("first-backend-key"),
					"INSTANA_DOWNLOAD_KEY": []byte("test-download-key"),
				},
				Type: corev1.SecretTypeOpaque,
			},
		},
		{
			name: "with UseSecretMounts false",
			agent: &instanav1.InstanaAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "test-namespace",
				},
				Spec: instanav1.InstanaAgentSpec{
					UseSecretMounts: pointer.To(false),
					Agent: instanav1.BaseAgentSpec{
						Key:           "test-key",
						DownloadKey:   "test-download-key",
						ProxyUser:     "proxy-user",
						ProxyPassword: "proxy-password",
					},
				},
			},
			backends: []backends.K8SensorBackend{
				{
					ResourceSuffix: "",
					EndpointKey:    "test-key",
				},
			},
			expectedSecret: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"key":         []byte("test-key"),
					"downloadKey": []byte("test-download-key"),
				},
				Type: corev1.SecretTypeOpaque,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertions := require.New(t)

			sb := NewSecretBuilder(tc.agent, tc.backends)
			actual := sb.Build()

			if tc.expectedSecret == nil {
				assertions.Empty(actual)
			} else {
				expected := optional.Of[client.Object](tc.expectedSecret)
				assertions.Equal(expected, actual)
			}
		})
	}
}
