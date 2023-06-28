package client

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/instana/instana-agent-operator/pkg/result"
)

func TestInstanaAgentClient_Apply(t *testing.T) {
	ctrl := gomock.NewController(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cm := corev1.ConfigMap{}
	opts := []k8sclient.PatchOption{k8sclient.DryRunAll}
	expectedErr := errors.New("awojsgeoisegoijsdg")

	mockK8sClient := NewMockClient(ctrl)
	mockK8sClient.EXPECT().Patch(
		gomock.Eq(ctx),
		gomock.Eq(&cm),
		gomock.Eq(k8sclient.Apply),
		gomock.Eq(append(opts, k8sclient.ForceOwnership, k8sclient.FieldOwner("instana-agent-operator"))),
	).Times(1).Return(expectedErr)

	client := instanaAgentClient{
		Client: mockK8sClient,
	}

	actualVal, actualErr := client.Apply(ctx, &cm, opts...).Get()

	assertions := require.New(t)

	assertions.Same(&cm, actualVal)
	assertions.Equal(expectedErr, actualErr)
}

func TestInstanaAgentClient_GetAsResult(t *testing.T) {
	assertions := require.New(t)
	ctrl := gomock.NewController(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	key := types.NamespacedName{
		Namespace: "adsf",
		Name:      "rasdgf",
	}
	obj := &unstructured.Unstructured{}
	opts := &k8sclient.GetOptions{
		Raw: &metav1.GetOptions{
			TypeMeta: metav1.TypeMeta{
				Kind:       "reoisoijd",
				APIVersion: "erifoijsd",
			},
			ResourceVersion: "adsfadsf",
		},
	}

	mockK8sClient := NewMockClient(ctrl)
	mockK8sClient.EXPECT().Get(
		gomock.Eq(ctx),
		gomock.Eq(key),
		gomock.Eq(obj),
		gomock.Eq(opts),
		gomock.Eq(opts),
	).Return(errors.New("foo"))

	client := instanaAgentClient{
		Client: mockK8sClient,
	}

	actual := client.GetAsResult(ctx, key, obj, opts, opts)
	assertions.Equal(result.Of[k8sclient.Object](obj, errors.New("foo")), actual)
}

func TestInstanaAgentClient_Exists(t *testing.T) {
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
				}, "some-resource",
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
					Name: "some-resource",
				}
				gvk := schema.GroupVersionKind{
					Group:   "somegroup",
					Version: "v1beta1",
					Kind:    "SomeKind",
				}

				obj := &unstructured.Unstructured{}
				obj.SetGroupVersionKind(gvk)

				k8sClient := NewMockClient(ctrl)
				k8sClient.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(key), gomock.Eq(obj)).Return(test.errOfGet)

				instanaClient := NewClient(k8sClient)

				actual := instanaClient.Exists(ctx, gvk, key)
				assertions.Equal(test.expected, actual)
			},
		)
	}
}

func expectExistsInvocation(client *MockClient, ctx context.Context, obj k8sclient.Object, shouldReturn error) {
	unstructuredObject := &unstructured.Unstructured{}
	unstructuredObject.SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())
	client.EXPECT().Get(
		gomock.Eq(ctx),
		gomock.Eq(k8sclient.ObjectKeyFromObject(obj)),
		gomock.Eq(unstructuredObject),
	).Return(shouldReturn)
}

func TestInstanaAgentClient_DeleteAllInTimeLimit(t *testing.T) {
	cmError := errors.New("cm")
	poError := errors.New("po")
	dsError := errors.New("ds")

	k8sObjectErrors := []error{cmError, poError, dsError}

	for _, test := range []struct {
		name           string
		clientBehavior func(client *MockClient, ctx context.Context, obj k8sclient.Object, i int)
		expectedErrors []error
	}{
		{
			name: "delete_errors",
			clientBehavior: func(client *MockClient, ctx context.Context, obj k8sclient.Object, i int) {
				client.EXPECT().Delete(gomock.Eq(ctx), gomock.Eq(obj)).Return(k8sObjectErrors[i])
			},
			expectedErrors: k8sObjectErrors,
		},
		{
			name: "verify_errors_until_timeout",
			clientBehavior: func(client *MockClient, ctx context.Context, obj k8sclient.Object, i int) {
				client.EXPECT().Delete(gomock.Eq(ctx), gomock.Eq(obj)).Return(nil)
				expectExistsInvocation(client, ctx, obj, k8sObjectErrors[i])
			},
			expectedErrors: []error{context.DeadlineExceeded},
		},
		{
			name: "objects_exist_until_timeout",
			clientBehavior: func(client *MockClient, ctx context.Context, obj k8sclient.Object, i int) {
				client.EXPECT().Delete(gomock.Eq(ctx), gomock.Eq(obj)).Return(nil)
				expectExistsInvocation(
					client, ctx, obj, k8sErrors.NewNotFound(
						schema.GroupResource{
							Group:    "apiextensions.k8s.io",
							Resource: "customresourcedefinitions",
						}, "some-resource",
					),
				)
			},
			expectedErrors: []error{context.DeadlineExceeded},
		},
		{
			name: "succeeds",
			clientBehavior: func(client *MockClient, ctx context.Context, obj k8sclient.Object, i int) {
				client.EXPECT().Delete(gomock.Eq(ctx), gomock.Eq(obj)).Return(nil)
				expectExistsInvocation(client, ctx, obj, nil)
			},
			expectedErrors: []error{nil},
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)
				ctrl := gomock.NewController(t)

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				expectedObjects := []k8sclient.Object{
					&corev1.ConfigMap{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "v1",
							Kind:       "ConfigMap",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "mycm",
							Namespace: "myns",
						},
					},
					&corev1.Pod{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "v1",
							Kind:       "Pod",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "mypod",
							Namespace: "myns",
						},
					},
					&appsv1.DaemonSet{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "appsv1",
							Kind:       "DaemonSet",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "myds",
							Namespace: "myns",
						},
					},
				}

				k8sClient := NewMockClient(ctrl)

				for i, obj := range expectedObjects {
					test.clientBehavior(k8sClient, ctx, obj, i)
				}

				instanaClient := &instanaAgentClient{
					Client: k8sClient,
				}

				actualObjects, actualError := instanaClient.DeleteAllInTimeLimit(
					ctx,
					expectedObjects,
					time.Second,
					time.Second,
				).Get()

				assertions.Equal(expectedObjects, actualObjects)

				for _, expectedErr := range test.expectedErrors {
					assertions.ErrorIs(actualError, expectedErr)
				}
			},
		)
	}
}
