/*
(c) Copyright IBM Corp. 2025

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

package mocks

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/namespaces"
	"github.com/instana/instana-agent-operator/pkg/result"
)

// MockInstanaAgentClient provides a testify mock implementation of InstanaAgentClient
type MockInstanaAgentClient struct {
	mock.Mock
}

func (m *MockInstanaAgentClient) Apply(
	ctx context.Context,
	obj client.Object,
	opts ...client.PatchOption,
) result.Result[client.Object] {
	args := m.Called(ctx, obj, opts)
	return args.Get(0).(result.Result[client.Object])
}

func (m *MockInstanaAgentClient) Exists(
	ctx context.Context,
	gvk schema.GroupVersionKind,
	key client.ObjectKey,
) result.Result[bool] {
	args := m.Called(ctx, gvk, key)
	return args.Get(0).(result.Result[bool])
}

func (m *MockInstanaAgentClient) DeleteAllInTimeLimit(
	ctx context.Context,
	objects []client.Object,
	timeout time.Duration,
	waitTime time.Duration,
	opts ...client.DeleteOption,
) result.Result[[]client.Object] {
	args := m.Called(ctx, objects, timeout, waitTime, opts)
	return args.Get(0).(result.Result[[]client.Object])
}

func (m *MockInstanaAgentClient) Get(
	ctx context.Context,
	key types.NamespacedName,
	obj client.Object,
	opts ...client.GetOption,
) error {
	args := m.Called(ctx, key, obj, opts)
	return args.Error(0)
}

func (m *MockInstanaAgentClient) List(
	ctx context.Context,
	list client.ObjectList,
	opts ...client.ListOption,
) error {
	args := m.Called(ctx, list, opts)
	return args.Error(0)
}

func (m *MockInstanaAgentClient) GetAsResult(
	ctx context.Context,
	key client.ObjectKey,
	obj client.Object,
	opts ...client.GetOption,
) result.Result[client.Object] {
	args := m.Called(ctx, key, obj, opts)
	return args.Get(0).(result.Result[client.Object])
}

func (m *MockInstanaAgentClient) Status() client.SubResourceWriter {
	args := m.Called()
	return args.Get(0).(client.SubResourceWriter)
}

func (m *MockInstanaAgentClient) Patch(
	ctx context.Context,
	obj client.Object,
	patch client.Patch,
	opts ...client.PatchOption,
) error {
	args := m.Called(ctx, obj, patch, opts)
	return args.Error(0)
}

func (m *MockInstanaAgentClient) Delete(
	ctx context.Context,
	obj client.Object,
	opts ...client.DeleteOption,
) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockInstanaAgentClient) GetNamespacesWithLabels(
	ctx context.Context,
) (namespaces.NamespacesDetails, error) {
	args := m.Called(ctx)
	return args.Get(0).(namespaces.NamespacesDetails), args.Error(1)
}
