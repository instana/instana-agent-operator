/*
(c) Copyright IBM Corp. 2024
(c) Copyright Instana Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/instana/instana-agent-operator/pkg/collections/list"
	"github.com/instana/instana-agent-operator/pkg/multierror"
	"github.com/instana/instana-agent-operator/pkg/result"

	k8sClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	FieldOwnerName = "instana-agent-operator"
)

func NewInstanaAgentClient(k8sClient k8sClient.Client) InstanaAgentClient {
	return &instanaAgentClient{
		k8sClient: k8sClient,
	}
}

type InstanaAgentClient interface {
	Apply(ctx context.Context, obj k8sClient.Object, opts ...k8sClient.PatchOption) result.Result[k8sClient.Object]
	Exists(ctx context.Context, gvk schema.GroupVersionKind, key k8sClient.ObjectKey) result.Result[bool]
	DeleteAllInTimeLimit(ctx context.Context, objects []k8sClient.Object, timeout time.Duration, waitTime time.Duration, opts ...k8sClient.DeleteOption) result.Result[[]k8sClient.Object]
	Get(ctx context.Context, key types.NamespacedName, obj k8sClient.Object, opts ...k8sClient.GetOption) error
	GetAsResult(ctx context.Context, key k8sClient.ObjectKey, obj k8sClient.Object, opts ...k8sClient.GetOption) result.Result[k8sClient.Object]
	Status() k8sClient.SubResourceWriter
	Patch(ctx context.Context, obj k8sClient.Object, patch k8sClient.Patch, opts ...k8sClient.PatchOption) error
	Delete(ctx context.Context, obj k8sClient.Object, opts ...k8sClient.DeleteOption) error
	GetNamespacesWithLabels(ctx context.Context) (map[string]map[string]string, error)
}

type instanaAgentClient struct {
	k8sClient k8sClient.Client
}

func (c *instanaAgentClient) DeleteAllInTimeLimit(
	ctx context.Context,
	objects []k8sClient.Object,
	timeout time.Duration,
	waitTime time.Duration,
	opts ...k8sClient.DeleteOption,
) result.Result[[]k8sClient.Object] {
	return result.Of(
		objects,
		c.deleteAllInTimeLimit(ctx, objects, timeout, waitTime, opts...),
	)
}

func (c *instanaAgentClient) Exists(
	ctx context.Context,
	gvk schema.GroupVersionKind,
	key k8sClient.ObjectKey,
) result.Result[bool] {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)

	res := c.GetAsResult(ctx, key, obj)

	return result.Map(res, wasRetrieved).Recover(ifNotFound)
}

func (c *instanaAgentClient) Get(
	ctx context.Context,
	key types.NamespacedName,
	obj k8sClient.Object,
	opts ...k8sClient.GetOption,
) error {
	return c.k8sClient.Get(ctx, key, obj, opts...)
}

func (c *instanaAgentClient) Patch(
	ctx context.Context,
	obj k8sClient.Object,
	patch k8sClient.Patch,
	opts ...k8sClient.PatchOption,
) error {
	return c.k8sClient.Patch(ctx, obj, patch, opts...)
}

func (c *instanaAgentClient) Apply(
	ctx context.Context,
	obj k8sClient.Object,
	opts ...k8sClient.PatchOption,
) result.Result[k8sClient.Object] {
	obj.SetManagedFields(nil)
	return result.Of(
		obj,
		c.k8sClient.Patch(
			ctx,
			obj,
			k8sClient.Apply,
			append(opts, k8sClient.ForceOwnership, k8sClient.FieldOwner(FieldOwnerName))...,
		),
	)
}

func (c *instanaAgentClient) Delete(
	ctx context.Context,
	obj k8sClient.Object,
	opts ...k8sClient.DeleteOption,
) error {
	return c.k8sClient.Delete(ctx, obj, opts...)
}

func (c *instanaAgentClient) GetAsResult(
	ctx context.Context,
	key k8sClient.ObjectKey,
	obj k8sClient.Object,
	opts ...k8sClient.GetOption,
) result.Result[k8sClient.Object] {
	return result.Of(obj, c.k8sClient.Get(ctx, key, obj, opts...))
}

func (c *instanaAgentClient) Status() k8sClient.SubResourceWriter {
	return c.k8sClient.Status()
}

func (c *instanaAgentClient) objectsExist(
	ctx context.Context,
	objects []k8sClient.Object,
) []result.Result[bool] {
	res := make([]result.Result[bool], 0, len(objects))

	for _, obj := range objects {
		objExistsRes := c.Exists(
			ctx,
			obj.GetObjectKind().GroupVersionKind(),
			k8sClient.ObjectKeyFromObject(obj),
		).OnFailure(
			func(err error) {
				log := logf.FromContext(ctx)
				log.Error(err, "failed to verify if resource has finished terminating", "Resource", obj)
			},
		)
		res = append(res, objExistsRes)
	}

	return res
}

func (c *instanaAgentClient) verifyDeletionStep(
	ctx context.Context,
	objects []k8sClient.Object,
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
	objects []k8sClient.Object,
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
	objects []k8sClient.Object,
	opts ...k8sClient.DeleteOption,
) error {
	errBuilder := multierror.NewMultiErrorBuilder()

	for _, obj := range objects {
		err := c.k8sClient.Delete(ctx, obj, opts...)
		errBuilder.Add(k8sClient.IgnoreNotFound(err))
	}

	return errBuilder.Build()
}

func (c *instanaAgentClient) deleteAllInTimeLimit(
	ctx context.Context,
	objects []k8sClient.Object,
	timeout time.Duration,
	waitTime time.Duration,
	opts ...k8sClient.DeleteOption,
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

func (c *instanaAgentClient) GetNamespacesWithLabels(
	ctx context.Context,
) (map[string]map[string]string, error) {
	namespaceLabelMap := make(map[string]map[string]string)
	namespaceList := &corev1.NamespaceList{}

	err := c.k8sClient.List(ctx, namespaceList)
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	result := make(map[string]map[string]string)
	for _, ns := range namespaceList.Items {
		labelsCopy := make(map[string]string)
		for k, v := range ns.Labels {
			labelsCopy[k] = v
		}
		result[ns.Name] = labelsCopy
	}
	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	fmt.Printf("%s", string(resultJSON))

	return namespaceLabelMap, nil
}
