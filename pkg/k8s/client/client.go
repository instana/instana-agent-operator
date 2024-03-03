package client

import (
	"context"
	"errors"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/instana/instana-agent-operator/pkg/collections/list"
	"github.com/instana/instana-agent-operator/pkg/multierror"
	"github.com/instana/instana-agent-operator/pkg/result"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	FieldOwnerName = "instana-agent-operator"
)

// ObjectResult alias is needed to workaround issues in mockgen
type ObjectResult = result.Result[k8sclient.Object]

// MultiObjectResult alias is needed to workaround issues in mockgen
type MultiObjectResult = result.Result[[]k8sclient.Object]

// BoolResult alias is needed to workaround issues in mockgen
type BoolResult = result.Result[bool]

type InstanaAgentClient interface {
	k8sclient.Client
	Apply(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.PatchOption) ObjectResult
	GetAsResult(
		ctx context.Context, key k8sclient.ObjectKey, obj k8sclient.Object, opts ...k8sclient.GetOption,
	) ObjectResult
	Exists(ctx context.Context, gvk schema.GroupVersionKind, key k8sclient.ObjectKey) BoolResult
	DeleteAllInTimeLimit(
		ctx context.Context,
		objects []k8sclient.Object,
		timeout time.Duration,
		waitTime time.Duration,
		opts ...k8sclient.DeleteOption,
	) MultiObjectResult
}

type instanaAgentClient struct {
	k8sclient.Client
}

func (c *instanaAgentClient) objectsExist(
	ctx context.Context,
	objects []k8sclient.Object,
) []BoolResult {
	res := make([]BoolResult, 0, len(objects))

	for _, obj := range objects {
		objExistsRes := c.Exists(ctx, obj.GetObjectKind().GroupVersionKind(), k8sclient.ObjectKeyFromObject(obj)).
			OnFailure(
				func(err error) {
					log := logf.FromContext(ctx)
					log.Error(err, "failed to verify if resource has finished terminating", "Resource", obj)
				},
			)
		res = append(res, objExistsRes)
	}

	return res
}

func doNotExist(res BoolResult) bool {
	return res.IsSuccess() && !res.ToOptional().Get()
}

func (c *instanaAgentClient) verifyDeletionStep(
	ctx context.Context,
	objects []k8sclient.Object,
	waitTime time.Duration,
) error {
	objectsExistResults := c.objectsExist(ctx, objects)

	switch list.NewConditions(objectsExistResults).All(doNotExist) {
	case true:
		return nil
	default:
		time.Sleep(waitTime)
		return c.verifyDeletion(ctx, objects, waitTime)
	}
}

func (c *instanaAgentClient) verifyDeletion(
	ctx context.Context,
	objects []k8sclient.Object,
	waitTime time.Duration,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return c.verifyDeletionStep(ctx, objects, waitTime)
	}
}

func (c *instanaAgentClient) deleteAll(
	ctx context.Context,
	objects []k8sclient.Object,
	opts ...k8sclient.DeleteOption,
) error {
	errBuilder := multierror.NewMultiErrorBuilder()

	for _, obj := range objects {
		err := c.Delete(ctx, obj, opts...)
		errBuilder.Add(k8sclient.IgnoreNotFound(err))
	}

	return errBuilder.Build()
}

func (c *instanaAgentClient) deleteAllInTimeLimit(
	ctx context.Context,
	objects []k8sclient.Object,
	timeout time.Duration,
	waitTime time.Duration,
	opts ...k8sclient.DeleteOption,
) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	switch err := c.deleteAll(ctx, objects, opts...); errors.Is(err, nil) {
	case true:
		return c.verifyDeletion(ctx, objects, waitTime)
	default:
		return err
	}
}

func (c *instanaAgentClient) DeleteAllInTimeLimit(
	ctx context.Context,
	objects []k8sclient.Object,
	timeout time.Duration,
	waitTime time.Duration,
	opts ...k8sclient.DeleteOption,
) MultiObjectResult {
	return result.Of(objects, c.deleteAllInTimeLimit(ctx, objects, timeout, waitTime, opts...))
}

func wasRetrieved(_ k8sclient.Object) result.Result[bool] {
	return result.OfSuccess(true)
}

func ifNotFound(err error) (bool, error) {
	return false, k8sclient.IgnoreNotFound(err)
}

func (c *instanaAgentClient) Exists(
	ctx context.Context,
	gvk schema.GroupVersionKind,
	key k8sclient.ObjectKey,
) BoolResult {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)

	res := c.GetAsResult(ctx, key, obj)

	return result.Map(res, wasRetrieved).Recover(ifNotFound)
}

func (c *instanaAgentClient) Apply(
	ctx context.Context, obj k8sclient.Object, opts ...k8sclient.PatchOption,
) result.Result[k8sclient.Object] {
	obj.SetManagedFields(nil)
	return result.Of(
		obj, c.Patch(
			ctx,
			obj,
			k8sclient.Apply,
			append(opts, k8sclient.ForceOwnership, k8sclient.FieldOwner(FieldOwnerName))...,
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
