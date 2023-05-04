package client

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/instana/instana-agent-operator/pkg/result"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ObjectResult alias is needed to workaround issues in mockgen
type ObjectResult = result.Result[k8sclient.Object]

// BoolResult alias is needed to workaround issues in mockgen
type BoolResult = result.Result[bool]

type InstanaAgentClient interface {
	k8sclient.Client
	Apply(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.PatchOption) ObjectResult
	GetAsResult(
		ctx context.Context, key k8sclient.ObjectKey, obj k8sclient.Object, opts ...k8sclient.GetOption,
	) ObjectResult
	Exists(ctx context.Context, gvk schema.GroupVersionKind, key k8sclient.ObjectKey) BoolResult
}

type instanaAgentClient struct {
	k8sclient.Client
}

func (c *instanaAgentClient) Exists(
	ctx context.Context,
	gvk schema.GroupVersionKind,
	key k8sclient.ObjectKey,
) BoolResult {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)

	res := c.GetAsResult(ctx, key, obj)

	return result.Map(
		res, func(_ k8sclient.Object) result.Result[bool] {
			return result.OfSuccess(true)
		},
	).Recover(
		func(err error) (bool, error) {
			return false, k8sclient.IgnoreNotFound(err)
		},
	)
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
