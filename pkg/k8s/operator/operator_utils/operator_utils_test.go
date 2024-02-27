package operator_utils

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/pointer"
	"github.com/instana/instana-agent-operator/pkg/result"
)

func testOperatorUtils_crdIsInstalled(t *testing.T, name string, methodName string) {
	for _, test := range []struct {
		name     string
		expected result.Result[bool]
	}{
		{
			name:     "crd_exists",
			expected: result.OfSuccess(true),
		},
		{
			name:     "crd_does_not_exist",
			expected: result.OfSuccess(false),
		},
		{
			name:     "error_getting_crd",
			expected: result.OfFailure[bool](errors.New("qwerty")),
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)
				ctrl := gomock.NewController(t)

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				gvk := schema.GroupVersionKind{
					Group:   "apiextensions.k8s.io",
					Version: "v1",
					Kind:    "CustomResourceDefinition",
				}
				key := types.NamespacedName{
					Name: name,
				}

				instanaClient := NewMockInstanaAgentClient(ctrl)
				instanaClient.EXPECT().Exists(gomock.Eq(ctx), gomock.Eq(gvk), gomock.Eq(key)).Return(test.expected)

				ot := NewOperatorUtils(ctx, instanaClient, &instanav1.InstanaAgent{})

				actual := reflect.ValueOf(ot).MethodByName(methodName).Call([]reflect.Value{})[0].Interface().(result.Result[bool])
				assertions.Equal(test.expected, actual)
			},
		)
	}
}

func TestOperatorUtils_ClusterIsOpenShift(t *testing.T) {
	t.Run(
		"user_provided", func(t *testing.T) {
			for _, test := range []struct {
				name            string
				userProvidedVal bool
				expected        result.Result[bool]
			}{
				{
					name:            "user_specifies_true",
					userProvidedVal: true,
					expected:        result.OfSuccess(true),
				},
				{
					name:            "user_specifies_false",
					userProvidedVal: false,
					expected:        result.OfSuccess(false),
				},
			} {
				t.Run(
					test.name, func(t *testing.T) {
						assertions := require.New(t)
						ctrl := gomock.NewController(t)

						ctx, cancel := context.WithCancel(context.Background())
						defer cancel()

						instanaClient := NewMockInstanaAgentClient(ctrl)

						ot := NewOperatorUtils(
							ctx, instanaClient, &instanav1.InstanaAgent{
								Spec: instanav1.InstanaAgentSpec{
									OpenShift: pointer.To(test.userProvidedVal),
								},
							},
						)

						actual := ot.ClusterIsOpenShift()
						assertions.Equal(test.expected, actual)
					},
				)
			}
		},
	)

	t.Run(
		"auto_detect", func(t *testing.T) {
			testOperatorUtils_crdIsInstalled(t, "clusteroperators.config.openshift.io", "ClusterIsOpenShift")
		},
	)
}
