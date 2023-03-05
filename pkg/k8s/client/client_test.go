package client

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestApply(t *testing.T) {
	ctrl := gomock.NewController(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cm := v1.ConfigMap{}
	opts := []k8sclient.PatchOption{k8sclient.DryRunAll}
	expected := errors.New("awojsgeoisegoijsdg")

	mockK8sClient := NewMockClient(ctrl)
	mockK8sClient.EXPECT().Patch(
		gomock.Eq(ctx),
		gomock.Eq(&cm),
		gomock.Eq(k8sclient.Apply),
		gomock.InAnyOrder(append(opts, k8sclient.ForceOwnership, k8sclient.FieldOwner("instana-agent-operator"))),
	).Times(1).Return(expected)

	mockTransformations := NewMockTransformations(ctrl)
	mockTransformations.EXPECT().AddCommonLabels(gomock.Eq(&cm)).Times(1)
	mockTransformations.EXPECT().AddOwnerReference(gomock.Eq(&cm)).Times(1)

	client := instanaAgentClient{
		Client:          mockK8sClient,
		Transformations: mockTransformations,
	}

	actual := client.Apply(ctx, &cm, opts...)

	assertions := require.New(t)

	assertions.Equal(expected, actual)
}
