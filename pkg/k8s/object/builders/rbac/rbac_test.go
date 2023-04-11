package rbac

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

func TestRbacBaseBuilder_Build(t *testing.T) {
	for _, isOpenShift := range []bool{true, false} {
		for _, rbacCreateSetByUser := range []bool{true, false} {
			t.Run(
				fmt.Sprintf("isOpenShift=%v_rbacCreateSetByUser=%v", isOpenShift, rbacCreateSetByUser),
				func(t *testing.T) {
					assertions := require.New(t)

					rbacBuilder := newRbacBuilder(
						&instanav1.InstanaAgent{
							Spec: instanav1.InstanaAgentSpec{
								Rbac: instanav1.Create{
									Create: rbacCreateSetByUser,
								},
							},
						}, isOpenShift, optional.BuilderFromLiteral[client.Object](&corev1.ConfigMap{}),
					)

					actual := rbacBuilder.Build()

					switch isOpenShift || rbacCreateSetByUser {
					case true:
						assertions.Equal(optional.Of[client.Object](&corev1.ConfigMap{}), actual)
					default:
						assertions.Empty(actual)
					}
				},
			)
		}
	}
}
