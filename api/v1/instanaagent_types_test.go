package v1

import (
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/instana/instana-agent-operator/pkg/pointer"
)

func TestInstanaAgent_Default(t *testing.T) {
	withOverrides := InstanaAgentSpec{
		Agent: BaseAgentSpec{
			EndpointHost: "abc",
			EndpointPort: "123",
			ExtendedImageSpec: ExtendedImageSpec{
				ImageSpec: ImageSpec{
					Name:       "icr.io/instana/asdf",
					Tag:        "1.1",
					PullPolicy: corev1.PullIfNotPresent,
				},
			},
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				Type: appsv1.OnDeleteDaemonSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDaemonSet{
					MaxUnavailable: pointer.To(intstr.FromInt(2)),
				},
			},
		},
		Rbac: Create{
			Create: pointer.To(false),
		},
		Service: Create{
			Create: pointer.To(false),
		},
		ServiceAccountSpec: ServiceAccountSpec{
			Create: Create{
				Create: pointer.To(false),
			},
		},
		K8sSensor: K8sSpec{
			ImageSpec: ImageSpec{
				Name:       "icr.io/instana/qwerty",
				Tag:        "2.2",
				PullPolicy: corev1.PullNever,
			},
			DeploymentSpec: KubernetesDeploymentSpec{
				Replicas: 2,
			},
		},
	}

	tests := []struct {
		name     string
		spec     *InstanaAgentSpec
		expected *InstanaAgentSpec
	}{
		{
			name: "no_user_overrides",
			spec: &InstanaAgentSpec{},
			expected: &InstanaAgentSpec{
				Agent: BaseAgentSpec{
					EndpointHost: "ingress-red-saas.instana.io",
					EndpointPort: "443",
					ExtendedImageSpec: ExtendedImageSpec{
						ImageSpec: ImageSpec{
							Name:       "icr.io/instana/agent",
							Tag:        "latest",
							PullPolicy: corev1.PullAlways,
						},
					},
					UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
						Type: appsv1.RollingUpdateDaemonSetStrategyType,
						RollingUpdate: &appsv1.RollingUpdateDaemonSet{
							MaxUnavailable: pointer.To(intstr.FromInt(1)),
						},
					},
				},
				Rbac: Create{
					Create: pointer.To(true),
				},
				Service: Create{
					Create: pointer.To(true),
				},
				ServiceAccountSpec: ServiceAccountSpec{
					Create: Create{
						Create: pointer.To(true),
					},
				},
				K8sSensor: K8sSpec{
					ImageSpec: ImageSpec{
						Name:       "icr.io/instana/k8sensor",
						Tag:        "latest",
						PullPolicy: corev1.PullAlways,
					},
					DeploymentSpec: KubernetesDeploymentSpec{
						Replicas: 1,
					},
				},
			},
		},
		{
			name:     "all_overrides",
			spec:     withOverrides.DeepCopy(),
			expected: withOverrides.DeepCopy(),
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				assertions := require.New(t)

				agent := &InstanaAgent{
					Spec: *tt.spec,
				}

				agent.Default()

				assertions.Equal(&InstanaAgent{Spec: *tt.expected}, agent)
			},
		)
	}
}
