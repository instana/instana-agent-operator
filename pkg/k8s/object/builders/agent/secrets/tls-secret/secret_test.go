package tls_secret

import (
	"fmt"
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

func TestSecretBuilder_Build(t *testing.T) {
	for _, secretName := range []string{"", rand.String(rand.IntnRange(1, 15))} {
		for _, key := range [][]byte{nil, []byte(rand.String(rand.IntnRange(1, 15)))} {
			for _, cert := range [][]byte{nil, []byte(rand.String(rand.IntnRange(1, 15)))} {
				t.Run(
					fmt.Sprintf(
						"%+v", struct {
							secretNameIsEmpty bool
							keyIsEmpty        bool
							certIsEmpty       bool
						}{
							secretNameIsEmpty: len(secretName) == 0,
							keyIsEmpty:        len(key) == 0,
							certIsEmpty:       len(cert) == 0,
						},
					), func(t *testing.T) {
						assertions := require.New(t)
						ctrl := gomock.NewController(t)

						namespace := rand.String(rand.IntnRange(1, 15))

						agent := &instanav1.InstanaAgent{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: namespace,
							},
							Spec: instanav1.InstanaAgentSpec{
								Agent: instanav1.BaseAgentSpec{
									TlsSpec: instanav1.TlsSpec{
										SecretName:  secretName,
										Key:         key,
										Certificate: cert,
									},
								},
							},
						}

						helpers := NewMockHelpers(ctrl)

						sb := &secretBuilder{
							InstanaAgent: agent,
							Helpers:      helpers,
						}

						switch {
						case secretName != "":
							fallthrough
						case len(key) == 0:
							fallthrough
						case len(cert) == 0:
							actual := sb.Build()
							assertions.Empty(actual)
						default:
							tlsSecretName := rand.String(rand.IntnRange(1, 15))

							helpers.EXPECT().TLSSecretName().Return(tlsSecretName)

							expected := optional.Of[client.Object](
								&corev1.Secret{
									TypeMeta: metav1.TypeMeta{
										APIVersion: "v1",
										Kind:       "Secret",
									},
									ObjectMeta: metav1.ObjectMeta{
										Name:      tlsSecretName,
										Namespace: namespace,
									},
									Data: map[string][]byte{
										corev1.TLSPrivateKeyKey: key,
										corev1.TLSCertKey:       cert,
									},
									Type: corev1.SecretTypeTLS,
								},
							)

							actual := sb.Build()

							assertions.Equal(expected, actual)
						}
					},
				)
			}
		}
	}
}
