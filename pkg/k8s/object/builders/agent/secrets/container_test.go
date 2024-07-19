/*
(c) Copyright IBM Corp. 2024
*/

package secrets

import (
	"testing"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSecretBuilderComponentName(t *testing.T) {
	s := NewContainerBuilder(&instanav1.InstanaAgent{}, &corev1.Secret{})

	assert.True(t, s.IsNamespaced())
}

func TestSecretBuilderIsNamespaced(t *testing.T) {
	s := NewContainerBuilder(&instanav1.InstanaAgent{}, &corev1.Secret{})

	assert.Equal(t, "instana-agent", s.ComponentName())
}

func TestContainerSecretBuilder(t *testing.T) {
	objectMeta := metav1.ObjectMeta{
		Name:      "object-name-value",
		Namespace: "object-namespace-value",
	}
	objectMetaConfig := metav1.ObjectMeta{
		Name:      objectMeta.Name + "-containers-instana-io",
		Namespace: objectMeta.Namespace,
	}
	secretType := metav1.TypeMeta{
		APIVersion: "v1",
		Kind:       "Secret",
	}

	for _, test := range []struct {
		name          string
		agentKey      string
		downloadKey   string
		imageSpecName string
		expectedData  map[string][]byte
		keysSecret    *corev1.Secret
	}{
		{
			name:          "Should return an empty client.Object when ImageSpec.Name does not contain instana registry",
			imageSpecName: "not-a-match",
			keysSecret:    &corev1.Secret{},
		},
		{
			name:          "Should return an empty client.Object when ImageSpec.Name contains instana registry but no keys were specified",
			imageSpecName: helpers.ContainersInstanaIORegistry,
			keysSecret:    &corev1.Secret{},
		},
		{
			name:          "Should return a secret when DownloadKey has been specified and ImageSpec.Name contains instana registry",
			imageSpecName: helpers.ContainersInstanaIORegistry,
			downloadKey:   "download-key",
			keysSecret:    &corev1.Secret{},
			expectedData: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte("{\"auths\":{\"containers.instana.io\":{\"auth\":\"Xzpkb3dubG9hZC1rZXk=\"}}}"),
			},
		},
		{
			name:          "Should return a secret when AgentKey has been specified and ImageSpec.Name contains instana registry",
			imageSpecName: helpers.ContainersInstanaIORegistry,
			agentKey:      "agent-key",
			keysSecret:    &corev1.Secret{},
			expectedData: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte("{\"auths\":{\"containers.instana.io\":{\"auth\":\"XzphZ2VudC1rZXk=\"}}}"),
			},
		},
		{
			name:          "Should return a secret when keysSecret v1.Secret passed over had downloadKey-field in its data and ImageSpec.Name contains instana registry",
			imageSpecName: helpers.ContainersInstanaIORegistry,
			keysSecret: &corev1.Secret{
				Type:       corev1.SecretTypeOpaque,
				TypeMeta:   secretType,
				ObjectMeta: objectMeta,
				Data: map[string][]byte{
					"downloadKey": []byte("download-key"),
				},
			},
			expectedData: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte("{\"auths\":{\"containers.instana.io\":{\"auth\":\"Xzpkb3dubG9hZC1rZXk=\"}}}"),
			},
		},
		{
			name:          "Should return a secret when keysSecret v1.Secret passed over had key-field in its data and ImageSpec.Name contains instana registry",
			imageSpecName: helpers.ContainersInstanaIORegistry,
			keysSecret: &corev1.Secret{
				Type:       corev1.SecretTypeOpaque,
				TypeMeta:   secretType,
				ObjectMeta: objectMeta,
				Data: map[string][]byte{
					"key": []byte("agent-key"),
				},
			},
			expectedData: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte("{\"auths\":{\"containers.instana.io\":{\"auth\":\"XzphZ2VudC1rZXk=\"}}}"),
			},
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				agent := &instanav1.InstanaAgent{
					ObjectMeta: objectMeta,
					Spec: instanav1.InstanaAgentSpec{
						Agent: instanav1.BaseAgentSpec{
							ExtendedImageSpec: instanav1.ExtendedImageSpec{
								// Only fetches container secrets when ImageSpec is "containers.instana.io"
								ImageSpec: instanav1.ImageSpec{Name: test.imageSpecName},
							},
							Key:         test.agentKey,
							DownloadKey: test.downloadKey,
						},
					},
				}

				sb := NewContainerBuilder(agent, test.keysSecret)

				actual := sb.Build()

				expected := &corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Secret",
					},
					ObjectMeta: objectMetaConfig,
					Data:       test.expectedData,
					Type:       corev1.SecretTypeDockerConfigJson,
				}

				if test.expectedData == nil {
					assert.Nil(t, actual.Get())
				} else {
					assert.Equal(t, expected, actual.Get())
				}

			},
		)
	}
}
