package helpers

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
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
							Create: true,
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
							Create: true,
						},
					},
				},
			},
			want: "-94jsdogijoijwgt",
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

func TestHelpers_KeysSecretName(t *testing.T) {
	t.Run(
		"keys_secret_not_provided_by_user", func(t *testing.T) {
			assertions := require.New(t)

			const expected = "riuoidfoisd"

			h := NewHelpers(
				&instanav1.InstanaAgent{
					ObjectMeta: metav1.ObjectMeta{
						Name: expected,
					},
				},
			)
			actual := h.KeysSecretName()

			assertions.Equal(expected, actual)
		},
	)
	t.Run(
		"keys_secret_is_provided_by_user", func(t *testing.T) {
			assertions := require.New(t)

			const expected = "riuoidfoisd"

			h := NewHelpers(
				&instanav1.InstanaAgent{
					ObjectMeta: metav1.ObjectMeta{
						Name: "oiew9oisdoijdsf",
					},
					Spec: instanav1.InstanaAgentSpec{
						Agent: instanav1.BaseAgentSpec{
							KeysSecret: expected,
						},
					},
				},
			)
			actual := h.KeysSecretName()

			assertions.Equal(expected, actual)
		},
	)
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
									Certificate: test.certificate,
									Key:         test.key,
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
	t.Run(
		"secret_name_set_explicitly", func(t *testing.T) {
			assertions := require.New(t)

			h := NewHelpers(
				&instanav1.InstanaAgent{
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
			)
			assertions.Equal("prpojdg", h.TLSSecretName())
		},
	)
	t.Run(
		"secret_name_not_set_explicitly", func(t *testing.T) {
			assertions := require.New(t)

			h := NewHelpers(
				&instanav1.InstanaAgent{
					ObjectMeta: metav1.ObjectMeta{
						Name: "risoijsdgljs",
					},
				},
			)
			assertions.Equal("risoijsdgljs-tls", h.TLSSecretName())
		},
	)
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
