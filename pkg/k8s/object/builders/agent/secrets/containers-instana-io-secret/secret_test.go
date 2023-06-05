package containers_instana_io_secret

import (
	"testing"

	"github.com/golang/mock/gomock"
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

func dockerConfigJsonForMarshal(password string) *DockerConfigJson {
	return &DockerConfigJson{
		Auths: map[string]DockerConfigAuth{
			"containers.instana.io": {
				Auth: []byte("_:" + password),
			},
		},
	}
}

func TestSecretBuilder_Build(t *testing.T) {
	randomNamespace := randString()
	randomAgentKey := randString()
	randomDownloadKey := randString()
	randomContainerSecretName := randString()
	randomMarshalResult := []byte(randString())

	expectedResult := optional.Of[client.Object](
		&corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      randomContainerSecretName,
				Namespace: randomNamespace,
			},
			Data: map[string][]byte{
				corev1.DockerConfigJsonKey: randomMarshalResult,
			},
			Type: corev1.SecretTypeDockerConfigJson,
		},
	)

	for _, test := range []struct {
		name               string
		useContainerSecret bool
		expectedPassword   string
		agentKey           string
		downloadKey        string
		expected           optional.Optional[client.Object]
	}{
		{
			name:               "should_be_empty",
			useContainerSecret: false,
			expected:           optional.Empty[client.Object](),
		},
		{
			name:               "download_key_is_specified",
			useContainerSecret: true,
			expectedPassword:   randomDownloadKey,
			agentKey:           randomAgentKey,
			downloadKey:        randomDownloadKey,
			expected:           expectedResult,
		},
		{
			name:               "download_key_is_not_specified",
			useContainerSecret: true,
			expectedPassword:   randomAgentKey,
			agentKey:           randomAgentKey,
			expected:           expectedResult,
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)
				ctrl := gomock.NewController(t)

				agent := &instanav1.InstanaAgent{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: randomNamespace,
					},
					Spec: instanav1.InstanaAgentSpec{
						Agent: instanav1.BaseAgentSpec{
							Key:         test.agentKey,
							DownloadKey: test.downloadKey,
						},
					},
				}

				helpers := NewMockHelpers(ctrl)
				helpers.EXPECT().UseContainersSecret().Return(test.useContainerSecret)
				if test.useContainerSecret {
					helpers.EXPECT().ContainersSecretName().Return(randomContainerSecretName)
				}

				marshaler := NewMockdockerConfigMarshaler(ctrl)
				if test.useContainerSecret {
					marshaler.EXPECT().MarshalOrDie(dockerConfigJsonForMarshal(test.expectedPassword)).Return(randomMarshalResult)
				}

				sb := &secretBuilder{
					InstanaAgent:          agent,
					Helpers:               helpers,
					dockerConfigMarshaler: marshaler,
				}

				actual := sb.Build()

				assertions.Equal(test.expected, actual)
			},
		)
	}
}
