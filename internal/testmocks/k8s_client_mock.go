/*
 * (c) Copyright IBM Corp. 2025
 */

package testmocks

import (
	"context"

	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// MockClient is a mock implementation of client.Client
type MockClient struct {
	mock.Mock
}

// Get mocks the Get method
func (m *MockClient) Get(
	ctx context.Context,
	key client.ObjectKey,
	obj client.Object,
	opts ...client.GetOption,
) error {
	args := m.Called(ctx, key, obj, opts)
	return args.Error(0)
}

// List mocks the List method
func (m *MockClient) List(
	ctx context.Context,
	list client.ObjectList,
	opts ...client.ListOption,
) error {
	args := m.Called(ctx, list, opts)
	return args.Error(0)
}

// Create mocks the Create method
func (m *MockClient) Create(
	ctx context.Context,
	obj client.Object,
	opts ...client.CreateOption,
) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

// Delete mocks the Delete method
func (m *MockClient) Delete(
	ctx context.Context,
	obj client.Object,
	opts ...client.DeleteOption,
) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

// Update mocks the Update method
func (m *MockClient) Update(
	ctx context.Context,
	obj client.Object,
	opts ...client.UpdateOption,
) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

// Patch mocks the Patch method
func (m *MockClient) Patch(
	ctx context.Context,
	obj client.Object,
	patch client.Patch,
	opts ...client.PatchOption,
) error {
	args := m.Called(ctx, obj, patch, opts)
	return args.Error(0)
}

// DeleteAllOf mocks the DeleteAllOf method
func (m *MockClient) DeleteAllOf(
	ctx context.Context,
	obj client.Object,
	opts ...client.DeleteAllOfOption,
) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

// Status mocks the Status method
func (m *MockClient) Status() client.StatusWriter {
	args := m.Called()
	return args.Get(0).(client.StatusWriter)
}

// Scheme mocks the Scheme method
func (m *MockClient) Scheme() *runtime.Scheme {
	args := m.Called()
	return args.Get(0).(*runtime.Scheme)
}

// RESTMapper mocks the RESTMapper method
func (m *MockClient) RESTMapper() meta.RESTMapper {
	args := m.Called()
	return args.Get(0).(meta.RESTMapper)
}

// SubResource mocks the SubResource method
func (m *MockClient) SubResource(subResource string) client.SubResourceClient {
	args := m.Called(subResource)
	return args.Get(0).(client.SubResourceClient)
}

// GroupVersionKindFor mocks the GroupVersionKindFor method
func (m *MockClient) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	args := m.Called(obj)
	return args.Get(0).(schema.GroupVersionKind), args.Error(1)
}

// IsObjectNamespaced mocks the IsObjectNamespaced method
func (m *MockClient) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	args := m.Called(obj)
	return args.Bool(0), args.Error(1)
}

// MockStatusWriter is a mock implementation of client.StatusWriter
type MockStatusWriter struct {
	mock.Mock
}

// Update mocks the Update method
func (m *MockStatusWriter) Update(
	ctx context.Context,
	obj client.Object,
	opts ...client.SubResourceUpdateOption,
) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

// Patch mocks the Patch method
func (m *MockStatusWriter) Patch(
	ctx context.Context,
	obj client.Object,
	patch client.Patch,
	opts ...client.SubResourcePatchOption,
) error {
	args := m.Called(ctx, obj, patch, opts)
	return args.Error(0)
}

// NewFakeClient creates a new fake client for testing
func NewFakeClient(initObjs ...client.Object) client.Client {
	return fake.NewClientBuilder().WithObjects(initObjs...).Build()
}

// Made with Bob
