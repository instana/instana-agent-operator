package helpers

import (
	"testing"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestServiceAccountName(t *testing.T) {
	t.Run("ServiceAccount name is set in spec", func(t *testing.T) {
		assertions := require.New(t)

		const expected = "0wegoijsdgo"

		actual := ServiceAccountName(&instanav1.InstanaAgent{
			Spec: instanav1.InstanaAgentSpec{
				ServiceAccountSpec: instanav1.ServiceAccountSpec{
					Name: instanav1.Name{
						Name: expected,
					},
				},
			},
		})

		assertions.Equal(expected, actual)
	})

	t.Run("ServiceAccount name is set in spec and create is true", func(t *testing.T) {
		assertions := require.New(t)

		const expected = "erhpoijsg94"

		actual := ServiceAccountName(&instanav1.InstanaAgent{
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
		})

		assertions.Equal(expected, actual)
	})

	t.Run("ServiceAccount create is true in spec", func(t *testing.T) {
		assertions := require.New(t)

		const expected = "-94jsdogijoijwgt"

		actual := ServiceAccountName(&instanav1.InstanaAgent{
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
		})

		assertions.Equal(expected, actual)
	})

	t.Run("No ServiceAccount options specified", func(t *testing.T) {
		assertions := require.New(t)

		const expected = "default"

		actual := ServiceAccountName(&instanav1.InstanaAgent{})

		assertions.Equal(expected, actual)
	})
}