package rbac

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
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
					ctrl := gomock.NewController(t)

					shouldCallChild := isOpenShift || rbacCreateSetByUser

					mockBuilder := NewMockObjectBuilder(ctrl)
					if shouldCallChild {
						mockBuilder.EXPECT().Build().Return(optional.Of[client.Object](&corev1.ConfigMap{}))
					}

					rbacBuilder := newRbacBuilder(
						&instanav1.InstanaAgent{
							Spec: instanav1.InstanaAgentSpec{
								Rbac: instanav1.Create{
									Create: rbacCreateSetByUser,
								},
							},
						}, isOpenShift, mockBuilder,
					)

					actual := rbacBuilder.Build()

					switch shouldCallChild {
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
