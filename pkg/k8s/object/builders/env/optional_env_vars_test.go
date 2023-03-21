package env

import (
	"testing"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/stretchr/testify/require"
)

// TODO: Generalize these?

func TestAgentModeEnv(t *testing.T) {
	t.Run("when_empty", func(t *testing.T) {
		assertions := require.New(t)
		actual := AgentModeEnv(&instanav1.InstanaAgent{}).Build()

		assertions.Equal(optional.Empty[corev1.EnvVar](), actual)
	})
	t.Run("with_value", func(t *testing.T) {
		assertions := require.New(t)
		actual := AgentModeEnv(&instanav1.InstanaAgent{
			Spec: instanav1.InstanaAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					Mode: instanav1.KUBERNETES,
				},
			},
		}).Build()

		assertions.Equal(
			optional.Of(corev1.EnvVar{
				Name:  "INSTANA_AGENT_MODE",
				Value: string(instanav1.KUBERNETES),
			}),
			actual,
		)
	})
}

func TestZoneNameEnv(t *testing.T) {
	t.Run("when_empty", func(t *testing.T) {
		assertions := require.New(t)
		actual := ZoneNameEnv(&instanav1.InstanaAgent{}).Build()

		assertions.Equal(optional.Empty[corev1.EnvVar](), actual)
	})
	t.Run("with_value", func(t *testing.T) {
		assertions := require.New(t)
		actual := ZoneNameEnv(&instanav1.InstanaAgent{
			Spec: instanav1.InstanaAgentSpec{
				Zone: instanav1.Name{
					Name: "oiweoiohewf",
				},
			},
		}).Build()

		assertions.Equal(
			optional.Of(corev1.EnvVar{
				Name:  "INSTANA_ZONE",
				Value: "oiweoiohewf",
			}),
			actual,
		)
	})
}

func TestClusterNameEnv(t *testing.T) {
	t.Run("when_empty", func(t *testing.T) {
		assertions := require.New(t)
		actual := ClusterNameEnv(&instanav1.InstanaAgent{}).Build()

		assertions.Equal(optional.Empty[corev1.EnvVar](), actual)
	})
	t.Run("with_value", func(t *testing.T) {
		assertions := require.New(t)
		actual := ClusterNameEnv(&instanav1.InstanaAgent{
			Spec: instanav1.InstanaAgentSpec{
				Cluster: instanav1.Name{
					Name: "oiweoiohewf",
				},
			},
		}).Build()

		assertions.Equal(
			optional.Of(corev1.EnvVar{
				Name:  "INSTANA_KUBERNETES_CLUSTER_NAME",
				Value: "oiweoiohewf",
			}),
			actual,
		)
	})
}
