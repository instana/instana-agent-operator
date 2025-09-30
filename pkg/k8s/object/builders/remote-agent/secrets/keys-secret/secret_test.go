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
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	backends "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/backends"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

func TestRemoteSecretBuilder_IsNamespaced_ComponentName(t *testing.T) {
	assertions := require.New(t)

	s := NewSecretBuilder(&instanav1.InstanaAgentRemote{}, make([]backends.RemoteSensorBackend, 0))

	assertions.True(s.IsNamespaced())
	assertions.Equal("instana-agent-remote", s.ComponentName())
}

func randString() string {
	return rand.String(rand.IntnRange(1, 15))
}

func emptyOrRandomString() []string {
	return []string{"", randString()}
}

func TestRemoteSecretBuilder_Build(t *testing.T) {
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

						agent := instanav1.InstanaAgentRemote{
							ObjectMeta: metav1.ObjectMeta{
								Name:      name,
								Namespace: namespace,
							},
							Spec: instanav1.InstanaAgentRemoteSpec{
								Agent: instanav1.BaseAgentSpec{
									KeysSecret:  keysSecret,
									Key:         key,
									DownloadKey: downloadKey,
								},
							},
						}

						backend := backends.NewRemoteSensorBackend("", key, downloadKey, "", "")
						var backends [1]backends.RemoteSensorBackend
						backends[0] = *backend

						sb := NewSecretBuilder(&agent, backends[:])

						actual := sb.Build()

						switch keysSecret {
						case "":
							data := make(map[string][]byte)

							if len(key) > 0 {
								data[constants.AgentKey] = []byte(key)
								data[constants.SecretFileAgentKey] = []byte(key)
							}

							if len(downloadKey) > 0 {
								data[constants.DownloadKey] = []byte(downloadKey)
								data[constants.SecretFileDownloadKey] = []byte(downloadKey)
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

func TestRemoteSecretBuilder_BuildIncludesProxySecretsWhenSecretMountsEnabled(t *testing.T) {
	assertions := require.New(t)

	agent := instanav1.InstanaAgentRemote{
		ObjectMeta: metav1.ObjectMeta{Name: "remote", Namespace: "instana"},
		Spec: instanav1.InstanaAgentRemoteSpec{
			Agent: instanav1.BaseAgentSpec{
				Key:           "agent-key",
				DownloadKey:   "download-key",
				ProxyHost:     "proxy.example.com",
				ProxyPort:     "3128",
				ProxyProtocol: "https",
				ProxyUser:     "proxyuser",
				ProxyPassword: "proxypass",
			},
		},
	}
	backend := backends.NewRemoteSensorBackend("", "agent-key", "download-key", "", "")
	secretOpt := NewSecretBuilder(&agent, []backends.RemoteSensorBackend{*backend}).Build()
	assertions.True(secretOpt.IsPresent())

	secret, ok := secretOpt.Get().(*corev1.Secret)
	assertions.True(ok)

	expectedHttps := "https://proxyuser:proxypass@proxy.example.com:3128"
	expected := map[string][]byte{
		constants.AgentKey:               []byte("agent-key"),
		constants.SecretFileAgentKey:     []byte("agent-key"),
		constants.DownloadKey:            []byte("download-key"),
		constants.SecretFileDownloadKey:  []byte("download-key"),
		constants.SecretKeyProxyUser:     []byte("proxyuser"),
		constants.SecretKeyProxyPassword: []byte("proxypass"),
		constants.SecretKeyHttpsProxy:    []byte(expectedHttps),
	}

	assertions.Equal(expected, secret.Data)
}

func TestRemoteSecretBuilder_BuildOmitsSecretMountDataWhenDisabled(t *testing.T) {
	assertions := require.New(t)
	useSecretMounts := pointer.To(false)

	agent := instanav1.InstanaAgentRemote{
		ObjectMeta: metav1.ObjectMeta{Name: "remote", Namespace: "instana"},
		Spec: instanav1.InstanaAgentRemoteSpec{
			UseSecretMounts: useSecretMounts,
			Agent: instanav1.BaseAgentSpec{
				Key:           "agent-key",
				DownloadKey:   "download-key",
				ProxyHost:     "proxy.example.com",
				ProxyPort:     "3128",
				ProxyProtocol: "https",
				ProxyUser:     "proxyuser",
				ProxyPassword: "proxypass",
			},
		},
	}
	backend := backends.NewRemoteSensorBackend("", "agent-key", "download-key", "", "")
	secretOpt := NewSecretBuilder(&agent, []backends.RemoteSensorBackend{*backend}).Build()
	assertions.True(secretOpt.IsPresent())

	secret, ok := secretOpt.Get().(*corev1.Secret)
	assertions.True(ok)

	expected := map[string][]byte{
		constants.AgentKey:    []byte("agent-key"),
		constants.DownloadKey: []byte("download-key"),
	}

	assertions.Equal(expected, secret.Data)
}
