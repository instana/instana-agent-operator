package helpers

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

func TestHelpers_ServiceAccountName(t *testing.T) {
	for _, tt := range []struct {
		name  string
		agent *instanav1.InstanaAgent
		want  string
	}{
		{
			name: "ServiceAccount name is set in spec",
			agent: &instanav1.InstanaAgent{
				Spec: instanav1.InstanaAgentSpec{
					ServiceAccountSpec: instanav1.ServiceAccountSpec{
						Name: instanav1.Name{
							Name: "0wegoijsdgo",
						},
					},
				},
			},
			want: "0wegoijsdgo",
		},
		{
			name: "ServiceAccount name is set in spec and create is true",
			agent: &instanav1.InstanaAgent{
				Spec: instanav1.InstanaAgentSpec{
					ServiceAccountSpec: instanav1.ServiceAccountSpec{
						Name: instanav1.Name{
							Name: "erhpoijsg94",
						},
						Create: instanav1.Create{
							Create: pointer.To(true),
						},
					},
				},
			},
			want: "erhpoijsg94",
		},
		{
			name: "ServiceAccount create is true in spec",
			agent: &instanav1.InstanaAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name: "-94jsdogijoijwgt",
				},
				Spec: instanav1.InstanaAgentSpec{
					ServiceAccountSpec: instanav1.ServiceAccountSpec{
						Create: instanav1.Create{
							Create: pointer.To(true),
						},
					},
				},
			},
			want: "-94jsdogijoijwgt",
		},
		{
			name: "ServiceAccount create is false in spec",
			agent: &instanav1.InstanaAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name: "-94jsdogijoijwgt",
				},
				Spec: instanav1.InstanaAgentSpec{
					ServiceAccountSpec: instanav1.ServiceAccountSpec{
						Create: instanav1.Create{
							Create: pointer.To(false),
						},
					},
				},
			},
			want: "default",
		},
		{
			name:  "No ServiceAccount options specified",
			agent: &instanav1.InstanaAgent{},
			want:  "default",
		},
	} {
		t.Run(
			tt.name, func(t *testing.T) {
				assertions := require.New(t)
				h := NewHelpers(tt.agent)
				assertions.Equal(tt.want, h.ServiceAccountName())
			},
		)
	}
}

func TestHelpers_TLSIsEnabled(t *testing.T) {
	for _, test := range []struct {
		name        string
		secretName  string
		certificate string
		key         string
		expected    bool
	}{
		{
			name: "all_empty",
		},
		{
			name:       "secret_name_filled",
			secretName: "adsfasg",
			expected:   true,
		},
		{
			name:       "secret_name_and_key_filled",
			secretName: "adsfasg",
			expected:   true,
			key:        "rgiosdoig",
		},
		{
			name:        "secret_name_and_cert_filled",
			secretName:  "adsfasg",
			expected:    true,
			certificate: "asoijegpoijsd",
		},
		{
			name:        "secret_name_cert_and_key_filled",
			secretName:  "adsfasg",
			expected:    true,
			certificate: "groijoijds",
			key:         "rwihjsdoijdsj",
		},
		{
			name:        "cert_filled",
			certificate: "woisoijdsjdsg",
		},
		{
			name: "key_filled",
			key:  "soihsoigjsdg",
		},
		{
			name:        "key_and_cert_filled",
			key:         "rwoihsdohjd",
			certificate: "ojoijsdoijoijdsf",
			expected:    true,
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)

				h := NewHelpers(
					&instanav1.InstanaAgent{
						Spec: instanav1.InstanaAgentSpec{
							Agent: instanav1.BaseAgentSpec{
								TlsSpec: instanav1.TlsSpec{
									SecretName:  test.secretName,
									Certificate: []byte(test.certificate),
									Key:         []byte(test.key),
								},
							},
						},
					},
				)
				assertions.Equal(test.expected, h.TLSIsEnabled())
			},
		)
	}
}

func TestHelpers_TLSSecretName(t *testing.T) {
	for _, tc := range []struct {
		name       string
		agent      *instanav1.InstanaAgent
		wantSecret string
	}{
		{
			name: "secret_name_set_explicitly",
			agent: &instanav1.InstanaAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name: "oioijsdjdsf",
				},
				Spec: instanav1.InstanaAgentSpec{
					Agent: instanav1.BaseAgentSpec{
						TlsSpec: instanav1.TlsSpec{
							SecretName: "prpojdg",
						},
					},
				},
			},
			wantSecret: "prpojdg",
		},
		{
			name: "secret_name_not_set_explicitly",
			agent: &instanav1.InstanaAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name: "risoijsdgljs",
				},
			},
			wantSecret: "risoijsdgljs-tls",
		},
	} {
		t.Run(
			tc.name, func(t *testing.T) {
				assertions := require.New(t)

				h := NewHelpers(tc.agent)

				gotSecret := h.TLSSecretName()

				assertions.Equal(tc.wantSecret, gotSecret)
			},
		)
	}
}

func TestHelpers_HeadlessServiceName(t *testing.T) {
	assertions := require.New(t)

	h := NewHelpers(
		&instanav1.InstanaAgent{
			ObjectMeta: metav1.ObjectMeta{
				Name: "rhjaoijdsoijoidsf",
			},
		},
	)
	assertions.Equal("rhjaoijdsoijoidsf-headless", h.HeadlessServiceName())
}

func TestHelpers_K8sSensorResourcesName(t *testing.T) {
	assertions := require.New(t)

	h := NewHelpers(
		&instanav1.InstanaAgent{
			ObjectMeta: metav1.ObjectMeta{
				Name: "rhjaoijdsoijoidsf",
			},
		},
	)
	assertions.Equal("rhjaoijdsoijoidsf-k8sensor", h.K8sSensorResourcesName())
}

func TestHelpers_ContainersSecretName(t *testing.T) {
	assertions := require.New(t)

	agentName := rand.String(rand.IntnRange(1, 15))

	h := NewHelpers(
		&instanav1.InstanaAgent{
			ObjectMeta: metav1.ObjectMeta{
				Name: agentName,
			},
		},
	)

	assertions.Equal(agentName+"-containers-instana-io", h.ContainersSecretName())
}

func TestHelpers_UseContainersSecret(t *testing.T) {
	for _, test := range []struct {
		name                    string
		userProvidedPullSecrets []corev1.LocalObjectReference
		imageName               string
		expected                bool
	}{
		{
			name:                    "nil_secrets_random_image",
			userProvidedPullSecrets: nil,
			imageName:               rand.String(rand.IntnRange(1, 15)),
			expected:                false,
		},
		{
			name:                    "empty_secrets_random_image",
			userProvidedPullSecrets: []corev1.LocalObjectReference{},
			imageName:               rand.String(rand.IntnRange(1, 15)),
			expected:                false,
		},
		{
			name: "non_empty_secrets_random_image",
			userProvidedPullSecrets: []corev1.LocalObjectReference{
				{
					Name: rand.String(rand.IntnRange(1, 15)),
				},
				{
					Name: rand.String(rand.IntnRange(1, 15)),
				},
			},
			imageName: rand.String(rand.IntnRange(1, 15)),
			expected:  false,
		},
		{
			name:                    "nil_secrets_image_has_prefix",
			userProvidedPullSecrets: nil,
			imageName:               "containers.instana.io/" + rand.String(rand.IntnRange(1, 15)),
			expected:                true,
		},
		{
			name:                    "empty_secrets_image_has_prefix",
			userProvidedPullSecrets: []corev1.LocalObjectReference{},
			imageName:               "containers.instana.io/" + rand.String(rand.IntnRange(1, 15)),
			expected:                false,
		},
		{
			name: "non_empty_secrets_image_has_prefix",
			userProvidedPullSecrets: []corev1.LocalObjectReference{
				{
					Name: rand.String(rand.IntnRange(1, 15)),
				},
				{
					Name: rand.String(rand.IntnRange(1, 15)),
				},
			},
			imageName: "containers.instana.io/" + rand.String(rand.IntnRange(1, 15)),
			expected:  false,
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)

				h := NewHelpers(
					&instanav1.InstanaAgent{
						ObjectMeta: metav1.ObjectMeta{
							Name: rand.String(rand.IntnRange(1, 15)),
						},
						Spec: instanav1.InstanaAgentSpec{
							Agent: instanav1.BaseAgentSpec{
								ExtendedImageSpec: instanav1.ExtendedImageSpec{
									PullSecrets: test.userProvidedPullSecrets,
									ImageSpec: instanav1.ImageSpec{
										Name: test.imageName,
									},
								},
							},
						},
					},
				)

				actual := h.UseContainersSecret()

				assertions.Equal(test.expected, actual)

				expectedSecrets := func() []corev1.LocalObjectReference {
					if test.expected {
						containersSecretName := h.ContainersSecretName()

						return []corev1.LocalObjectReference{
							{
								Name: containersSecretName,
							},
						}
					} else {
						return test.userProvidedPullSecrets
					}
				}()

				actualSecrets := h.ImagePullSecrets()

				assertions.Equal(expectedSecrets, actualSecrets)
			},
		)
	}
}
