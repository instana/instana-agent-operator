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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/optional"
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

func TestOperatorUtils_ApplyAll(t *testing.T) {
	cmError := errors.New("cm")
	poError := errors.New("po")
	dsError := errors.New("ds")

	k8sObjectErrors := []error{cmError, poError, dsError}

	lifecycleError := errors.New("lifecycle")

	for _, test := range []struct {
		name                string
		clientBehavior      func(ctx context.Context, client *MockInstanaAgentClient, obj k8sclient.Object, i int)
		lifecycleBehavior   func(lifecycle *MockDependentLifecycleManager, expectedObjects []k8sclient.Object)
		expectedErrors      []error
		shouldReturnObjects bool
	}{
		{
			name: "dry_run_errors",
			clientBehavior: func(ctx context.Context, client *MockInstanaAgentClient, obj k8sclient.Object, i int) {
				client.EXPECT().Apply(
					ctx,
					obj,
					k8sclient.DryRunAll,
				).Return(result.OfFailure[k8sclient.Object](k8sObjectErrors[i]))
			},
			lifecycleBehavior: func(lifecycle *MockDependentLifecycleManager, expectedObjects []k8sclient.Object) {},
			expectedErrors:    k8sObjectErrors,
		},
		{
			name: "update_lifecycle_cm_errors",
			clientBehavior: func(ctx context.Context, client *MockInstanaAgentClient, obj k8sclient.Object, i int) {
				client.EXPECT().Apply(
					ctx,
					obj,
					k8sclient.DryRunAll,
				).Return(result.OfSuccess[k8sclient.Object](nil))
			},
			lifecycleBehavior: func(lifecycle *MockDependentLifecycleManager, expectedObjects []k8sclient.Object) {
				lifecycle.EXPECT().UpdateDependentLifecycleInfo(gomock.Eq(expectedObjects)).Return(
					result.Of(
						expectedObjects,
						lifecycleError,
					),
				)
			},
			expectedErrors: []error{lifecycleError},
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
				).Return(result.OfFailure[k8sclient.Object](k8sObjectErrors[i]))
			},
			lifecycleBehavior: func(lifecycle *MockDependentLifecycleManager, expectedObjects []k8sclient.Object) {
				lifecycle.EXPECT().UpdateDependentLifecycleInfo(gomock.Eq(expectedObjects)).Return(result.OfSuccess(expectedObjects))
			},
			expectedErrors: k8sObjectErrors,
		},
		{
			name: "delete_orphaned_dependents_errors",
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
			lifecycleBehavior: func(lifecycle *MockDependentLifecycleManager, expectedObjects []k8sclient.Object) {
				lifecycle.EXPECT().UpdateDependentLifecycleInfo(gomock.Eq(expectedObjects)).Return(result.OfSuccess(expectedObjects))
				lifecycle.EXPECT().DeleteOrphanedDependents(gomock.Eq(expectedObjects)).Return(
					result.Of(
						expectedObjects,
						lifecycleError,
					),
				)
			},
			expectedErrors: []error{lifecycleError},
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
			lifecycleBehavior: func(lifecycle *MockDependentLifecycleManager, expectedObjects []k8sclient.Object) {
				lifecycle.EXPECT().UpdateDependentLifecycleInfo(gomock.Eq(expectedObjects)).Return(result.OfSuccess(expectedObjects))
				lifecycle.EXPECT().DeleteOrphanedDependents(gomock.Eq(expectedObjects)).Return(result.OfSuccess(expectedObjects))
			},
			expectedErrors:      []error{nil},
			shouldReturnObjects: true,
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

				mockDependentLifecycleManager := NewMockDependentLifecycleManager(ctrl)
				test.lifecycleBehavior(mockDependentLifecycleManager, expectedObjects)

				ot := &operatorUtils{
					ctx:                       ctx,
					InstanaAgentClient:        client,
					InstanaAgent:              nil,
					builderTransformer:        builderTransformer,
					DependentLifecycleManager: mockDependentLifecycleManager,
				}

				actualObjects, actualError := ot.ApplyAll(mockBuilders...).Get()

				if test.shouldReturnObjects {
					assertions.Equal(expectedObjects, actualObjects)
				}

				for _, expectedErr := range test.expectedErrors {
					assertions.ErrorIs(actualError, expectedErr)
				}
			},
		)
	}
}
