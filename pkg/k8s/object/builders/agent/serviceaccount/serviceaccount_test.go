package serviceaccount

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
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

func TestServiceAccountBuilder_IsNamespaced_ComponentName(t *testing.T) {
	assertions := require.New(t)

	sb := NewServiceAccountBuilder(nil)

	assertions.True(sb.IsNamespaced())
	assertions.Equal(constants.ComponentInstanaAgent, sb.ComponentName())
}

func TestServiceAccountBuilder_Build(t *testing.T) {
	for _, test := range []struct {
		createServiceAccount *bool
	}{
		{
			createServiceAccount: pointer.To(true),
		},
		{
			createServiceAccount: pointer.To(false),
		},
		{
			createServiceAccount: nil,
		},
	} {
		t.Run(
			fmt.Sprintf("%+v", test), func(t *testing.T) {
				assertions := require.New(t)
				ctrl := gomock.NewController(t)

				serviceAccountName := rand.String(10)
				namespace := rand.String(10)

				agent := &instanav1.InstanaAgent{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespace,
					},
					Spec: instanav1.InstanaAgentSpec{
						ServiceAccountSpec: instanav1.ServiceAccountSpec{
							Create: instanav1.Create{
								Create: test.createServiceAccount,
							},
						},
					},
				}

				expected := optional.Of[client.Object](
					&corev1.ServiceAccount{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "v1",
							Kind:       "ServiceAccount",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      serviceAccountName,
							Namespace: namespace,
						},
					},
				)

				helpers := NewMockHelpers(ctrl)
				if pointer.DerefOrEmpty(test.createServiceAccount) {
					helpers.EXPECT().ServiceAccountName().Return(serviceAccountName)
				}

				sb := &serviceAccountBuilder{
					InstanaAgent: agent,
					Helpers:      helpers,
				}

				actual := sb.Build()

				if pointer.DerefOrEmpty(test.createServiceAccount) {
					assertions.Equal(expected, actual)
				} else {
					assertions.Empty(actual)
				}
			},
		)
	}
}
