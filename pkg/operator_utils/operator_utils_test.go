package operator_utils

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/pointer"
	"github.com/instana/instana-agent-operator/pkg/result"
)

func testOperatorUtils_crdIsInstalled(t *testing.T, name string, methodName string) {
	for _, test := range []struct {
		name     string
		errOfGet error
		expected result.Result[bool]
	}{
		{
			name:     "exists",
			errOfGet: nil,
			expected: result.OfSuccess(true),
		},
		{
			name: "not_exists",
			errOfGet: k8sErrors.NewNotFound(
				schema.GroupResource{
					Group:    "apiextensions.k8s.io",
					Resource: "customresourcedefinitions",
				}, name,
			),
			expected: result.OfSuccess(false),
		},
		{
			name:     "error",
			errOfGet: errors.New("qwerty"),
			expected: result.OfFailure[bool](errors.New("qwerty")),
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)
				ctrl := gomock.NewController(t)

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				key := types.NamespacedName{
					Name: name,
				}
				obj := &unstructured.Unstructured{}
				obj.SetAPIVersion("apiextensions.k8s.io/v1")
				obj.SetKind("CustomResourceDefinition")

				instanaClient := NewMockInstanaAgentClient(ctrl)
				instanaClient.EXPECT().GetAsResult(
					gomock.Eq(ctx),
					gomock.Eq(key),
					gomock.Eq(obj),
				).Return(result.Of[k8sclient.Object](obj, test.errOfGet))

				ot := NewOperatorUtils(ctx, instanaClient, &instanav1.InstanaAgent{})

				actual := reflect.ValueOf(ot).MethodByName(methodName).Call([]reflect.Value{})[0].Interface().(result.Result[bool])
				assertions.Equal(test.expected, actual)
			},
		)
	}
}

func TestOperatorUtils_ClusterIsOpenShift(t *testing.T) {
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

	testOperatorUtils_crdIsInstalled(t, "clusteroperators.config.openshift.io", "ClusterIsOpenShift")
}
