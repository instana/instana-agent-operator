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

	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MockClient provides a testify mock implementation of the controller-runtime client.Client interface
type MockClient struct {
	mock.Mock
}

func (m *MockClient) Get(
	ctx context.Context,
	key types.NamespacedName,
	obj client.Object,
	opts ...client.GetOption,
) error {
	args := m.Called(ctx, key, obj, opts)
	return args.Error(0)
}

func (m *MockClient) List(
	ctx context.Context,
	list client.ObjectList,
	opts ...client.ListOption,
) error {
	args := m.Called(ctx, list, opts)
	return args.Error(0)
}

func (m *MockClient) Create(
	ctx context.Context,
	obj client.Object,
	opts ...client.CreateOption,
) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Delete(
	ctx context.Context,
	obj client.Object,
	opts ...client.DeleteOption,
) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Update(
	ctx context.Context,
	obj client.Object,
	opts ...client.UpdateOption,
) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Patch(
	ctx context.Context,
	obj client.Object,
	patch client.Patch,
	opts ...client.PatchOption,
) error {
	args := m.Called(ctx, obj, patch, opts)
	return args.Error(0)
}

func (m *MockClient) DeleteAllOf(
	ctx context.Context,
	obj client.Object,
	opts ...client.DeleteAllOfOption,
) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Status() client.SubResourceWriter {
	args := m.Called()
	return args.Get(0).(client.SubResourceWriter)
}

func (m *MockClient) SubResource(subResource string) client.SubResourceClient {
	args := m.Called(subResource)
	return args.Get(0).(client.SubResourceClient)
}

func (m *MockClient) Scheme() *runtime.Scheme {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*runtime.Scheme)
}

func (m *MockClient) RESTMapper() meta.RESTMapper {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(meta.RESTMapper)
}

func (m *MockClient) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	args := m.Called(obj)
	return args.Get(0).(schema.GroupVersionKind), args.Error(1)
}

func (m *MockClient) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	args := m.Called(obj)
	return args.Bool(0), args.Error(1)
}

func (m *MockClient) Apply(
	ctx context.Context,
	obj runtime.ApplyConfiguration,
	opts ...client.ApplyOption,
) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

// MockSubResourceWriter provides a testify mock implementation of client.SubResourceWriter
type MockSubResourceWriter struct {
	mock.Mock
}

func (m *MockSubResourceWriter) Create(
	ctx context.Context,
	obj client.Object,
	subResource client.Object,
	opts ...client.SubResourceCreateOption,
) error {
	args := m.Called(ctx, obj, subResource, opts)
	return args.Error(0)
}

func (m *MockSubResourceWriter) Update(
	ctx context.Context,
	obj client.Object,
	opts ...client.SubResourceUpdateOption,
) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockSubResourceWriter) Patch(
	ctx context.Context,
	obj client.Object,
	patch client.Patch,
	opts ...client.SubResourcePatchOption,
) error {
	args := m.Called(ctx, obj, patch, opts)
	return args.Error(0)
}
