package helpers

import (
	"testing"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
)

func TestServiceAccountName(t *testing.T) {

	t.Run(
		"ServiceAccount name is set in spec", func(t *testing.T) {
			assertions := require.New(t)

			const expected = "0wegoijsdgo"

			h := NewHelpers(
				&instanav1.InstanaAgent{
					Spec: instanav1.InstanaAgentSpec{
						ServiceAccountSpec: instanav1.ServiceAccountSpec{
							Name: instanav1.Name{
								Name: expected,
							},
						},
					},
				},
			)

			assertions.Equal(expected, h.ServiceAccountName())
		},
	)

	t.Run(
		"ServiceAccount name is set in spec and create is true", func(t *testing.T) {
			assertions := require.New(t)

			const expected = "erhpoijsg94"

			h := NewHelpers(
				&instanav1.InstanaAgent{
					Spec: instanav1.InstanaAgentSpec{
						ServiceAccountSpec: instanav1.ServiceAccountSpec{
							Name: instanav1.Name{
								Name: expected,
							},
							Create: instanav1.Create{
								Create: true,
							},
						},
					},
				},
			)

			assertions.Equal(expected, h.ServiceAccountName())
		},
	)

	t.Run(
		"ServiceAccount create is true in spec", func(t *testing.T) {
			assertions := require.New(t)

			const expected = "-94jsdogijoijwgt"

			h := NewHelpers(
				&instanav1.InstanaAgent{
					ObjectMeta: v1.ObjectMeta{
						Name: expected,
					},
					Spec: instanav1.InstanaAgentSpec{
						ServiceAccountSpec: instanav1.ServiceAccountSpec{
							Create: instanav1.Create{
								Create: true,
							},
						},
					},
				},
			)

			assertions.Equal(expected, h.ServiceAccountName())
		},
	)

	t.Run(
		"No ServiceAccount options specified", func(t *testing.T) {
			assertions := require.New(t)

			const expected = "default"

			h := NewHelpers(&instanav1.InstanaAgent{})

			assertions.Equal(expected, h.ServiceAccountName())
		},
	)
}

func TestHelpers_KeysSecretName(t *testing.T) {
	t.Run(
		"keys_secret_not_provided_by_user", func(t *testing.T) {
			assertions := require.New(t)

			const expected = "riuoidfoisd"

			h := NewHelpers(
				&instanav1.InstanaAgent{
					ObjectMeta: v1.ObjectMeta{
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
					ObjectMeta: v1.ObjectMeta{
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
					ObjectMeta: v1.ObjectMeta{
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
					ObjectMeta: v1.ObjectMeta{
						Name: "risoijsdgljs",
					},
				},
			)
			assertions.Equal("risoijsdgljs-tls", h.TLSSecretName())
		},
	)
}
