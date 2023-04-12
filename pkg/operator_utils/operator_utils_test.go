package operator_utils

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/builder"
	"github.com/instana/instana-agent-operator/pkg/optional"
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
			name:     "crd_exists",
			errOfGet: nil,
			expected: result.OfSuccess(true),
		},
		{
			name: "crd_does_not_exist",
			errOfGet: k8sErrors.NewNotFound(
				schema.GroupResource{
					Group:    "apiextensions.k8s.io",
					Resource: "customresourcedefinitions",
				}, name,
			),
			expected: result.OfSuccess(false),
		},
		{
			name:     "error_getting_crd",
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

func TestOperatorUtils_ApplyAll(t *testing.T) {
	cmError := errors.New("cm")
	poError := errors.New("po")
	dsError := errors.New("ds")

	allErrors := []error{cmError, poError, dsError}

	for _, test := range []struct {
		name           string
		clientBehavior func(ctx context.Context, client *MockInstanaAgentClient, obj k8sclient.Object, i int)
		expectedErrors []error
	}{
		{
			name: "dry_run_errors",
			clientBehavior: func(ctx context.Context, client *MockInstanaAgentClient, obj k8sclient.Object, i int) {
				client.EXPECT().Apply(
					ctx,
					obj,
					k8sclient.DryRunAll,
				).Return(result.OfFailure[k8sclient.Object](allErrors[i]))
			},
			expectedErrors: allErrors,
		},
		{
			name: "cluster_persist_errors",
			clientBehavior: func(ctx context.Context, client *MockInstanaAgentClient, obj k8sclient.Object, i int) {
				client.EXPECT().Apply(
					ctx,
					obj,
					k8sclient.DryRunAll,
				).Return(result.OfSuccess[k8sclient.Object](nil))
				client.EXPECT().Apply(
					ctx,
					obj,
				).Return(result.OfFailure[k8sclient.Object](allErrors[i]))
			},
			expectedErrors: allErrors,
		},
		{
			name: "succeeds",
			clientBehavior: func(ctx context.Context, client *MockInstanaAgentClient, obj k8sclient.Object, i int) {
				client.EXPECT().Apply(
					ctx,
					obj,
					k8sclient.DryRunAll,
				).Return(result.OfSuccess[k8sclient.Object](nil))
				client.EXPECT().Apply(
					ctx,
					obj,
				).Return(result.OfSuccess[k8sclient.Object](nil))
			},
			expectedErrors: []error{nil},
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)
				ctrl := gomock.NewController(t)

				expectedObjects := []k8sclient.Object{
					&corev1.ConfigMap{},
					&corev1.Pod{},
					&appsv1.DaemonSet{},
				}

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				client := NewMockInstanaAgentClient(ctrl)
				for i, obj := range expectedObjects {
					test.clientBehavior(ctx, client, obj, i)
				}

				mockBuilders := make([]builder.ObjectBuilder, 0, len(expectedObjects))
				for range expectedObjects {
					mockBuilders = append(mockBuilders, NewMockObjectBuilder(ctrl))
				}

				builderTransformer := NewMockBuilderTransformer(ctrl)
				for i, mockBuilder := range mockBuilders {
					builderTransformer.EXPECT().Apply(mockBuilder).Return(optional.Of[k8sclient.Object](expectedObjects[i]))
				}

				ot := &operatorUtils{
					ctx:                ctx,
					InstanaAgentClient: client,
					InstanaAgent:       nil,
					builderTransformer: builderTransformer,
				}

				actualObjects, actualError := ot.ApplyAll(mockBuilders).Get()
				assertions.Equal(expectedObjects, actualObjects)
				for _, expectedErr := range test.expectedErrors {
					assertions.ErrorIs(actualError, expectedErr)
				}
			},
		)
	}
}
