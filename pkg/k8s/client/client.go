package client

import (
	"context"

	"github.com/instana/instana-agent-operator/pkg/result"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"

	"github.com/instana/instana-agent-operator/pkg/k8s/object/transformations"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type InstanaAgentClient interface {
	k8sclient.Client
	Apply(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.PatchOption) result.Result[k8sclient.Object]
	GetAsResult(
		ctx context.Context, key k8sclient.ObjectKey, obj k8sclient.Object, opts ...k8sclient.GetOption,
	) result.Result[k8sclient.Object]
}

type instanaAgentClient struct {
	k8sclient.Client
	transformations.Transformations
}

func (c *instanaAgentClient) Apply(
	ctx context.Context, obj k8sclient.Object, opts ...k8sclient.PatchOption,
) result.Result[k8sclient.Object] {
	c.AddCommonLabels(obj)
	c.AddOwnerReference(obj)

	return result.Of(
		obj, c.Patch(
			ctx,
			obj,
			k8sclient.Apply,
			append(opts, k8sclient.ForceOwnership, k8sclient.FieldOwner("instana-agent-operator"))...,
		),
	)
}

// TODO: test

func (c *instanaAgentClient) GetAsResult(
	ctx context.Context, key k8sclient.ObjectKey, obj k8sclient.Object, opts ...k8sclient.GetOption,
) result.Result[k8sclient.Object] {
	return result.Of(obj, c.Client.Get(ctx, key, obj, opts...))
}

func NewClient(k8sClient k8sclient.Client, agent *instanav1.InstanaAgent) InstanaAgentClient {
	return &instanaAgentClient{
		Client:          k8sClient,
		Transformations: transformations.NewTransformations(agent),
	}
}
