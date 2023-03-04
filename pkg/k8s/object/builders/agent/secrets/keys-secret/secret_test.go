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
	"github.com/instana/instana-agent-operator/pkg/optional"
)

func TestSecretBuilder_IsNamespaced_ComponentName(t *testing.T) {
	assertions := require.New(t)

	s := NewSecretBuilder(&instanav1.InstanaAgent{})

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

						sb := NewSecretBuilder(&agent)

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
