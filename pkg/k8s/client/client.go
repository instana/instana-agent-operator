package client

import (
	"context"

	"github.com/instana/instana-agent-operator/pkg/k8s/object/transformations"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type InstanaAgentClient interface {
	k8sclient.Client
	Apply(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.PatchOption) error
}

type instanaAgentClient struct {
	k8sclient.Client
	transformations.Transformations
}

func (c *instanaAgentClient) Apply(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.PatchOption) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	c.AddCommonLabels(obj)
	c.AddOwnerReference(obj)

	return c.Patch(
		ctx,
		obj,
		k8sclient.Apply,
		append(opts, k8sclient.ForceOwnership, k8sclient.FieldOwner("instana-agent-operator"))...,
	)
}

func NewClient(k8sClient k8sclient.Client) InstanaAgentClient {
	return &instanaAgentClient{
		Client: k8sClient,
	}
}
