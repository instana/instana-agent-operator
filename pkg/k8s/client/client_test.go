package client

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
