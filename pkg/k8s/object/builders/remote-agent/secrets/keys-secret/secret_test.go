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
	"github.com/instana/instana-agent-operator/pkg/optional"
)

func TestRemoteSecretBuilder_IsNamespaced_ComponentName(t *testing.T) {
	assertions := require.New(t)

	s := NewSecretBuilder(&instanav1.RemoteAgent{}, make([]backends.K8SensorBackend, 0))

	assertions.True(s.IsNamespaced())
	assertions.Equal("remote-instana-agent", s.ComponentName())
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

						agent := instanav1.RemoteAgent{
							ObjectMeta: metav1.ObjectMeta{
								Name:      name,
								Namespace: namespace,
							},
							Spec: instanav1.RemoteAgentSpec{
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
