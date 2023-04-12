package client

import (
	"context"

	"github.com/instana/instana-agent-operator/pkg/result"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ObjectResult alias is needed to workaround issues in mockgen
type ObjectResult = result.Result[k8sclient.Object]

type InstanaAgentClient interface {
	k8sclient.Client
	Apply(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.PatchOption) ObjectResult
	GetAsResult(
		ctx context.Context, key k8sclient.ObjectKey, obj k8sclient.Object, opts ...k8sclient.GetOption,
	) ObjectResult
	// TODO: Delete Collection
}

type instanaAgentClient struct {
	k8sclient.Client
}

func (c *instanaAgentClient) Apply(
	ctx context.Context, obj k8sclient.Object, opts ...k8sclient.PatchOption,
) result.Result[k8sclient.Object] {
	return result.Of(
		obj, c.Patch(
			ctx,
			obj,
			k8sclient.Apply,
			append(opts, k8sclient.ForceOwnership, k8sclient.FieldOwner("instana-agent-operator"))...,
		),
	)
}

func (c *instanaAgentClient) GetAsResult(
	ctx context.Context, key k8sclient.ObjectKey, obj k8sclient.Object, opts ...k8sclient.GetOption,
) result.Result[k8sclient.Object] {
	return result.Of(obj, c.Client.Get(ctx, key, obj, opts...))
}

func NewClient(k8sClient k8sclient.Client) InstanaAgentClient {
	return &instanaAgentClient{
		Client: k8sClient,
	}
}
