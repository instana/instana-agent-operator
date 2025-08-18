/*
 * (c) Copyright IBM Corp. 2025
 */

package testmocks

import (
	"context"
	"time"

	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/namespaces"
	"github.com/instana/instana-agent-operator/pkg/result"
	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MockInstanaAgentClient is a mock implementation of InstanaAgentClient interface
type MockInstanaAgentClient struct {
	mock.Mock
}

// Apply mocks the Apply method
func (m *MockInstanaAgentClient) Apply(
	ctx context.Context,
	obj client.Object,
	opts ...client.PatchOption,
) result.Result[client.Object] {
	args := m.Called(append([]interface{}{ctx, obj}, mock.Anything)...)
	return args.Get(0).(result.Result[client.Object])
}

// Exists mocks the Exists method
func (m *MockInstanaAgentClient) Exists(
	ctx context.Context,
	gvk schema.GroupVersionKind,
	key client.ObjectKey,
) result.Result[bool] {
	args := m.Called(ctx, gvk, key)
	return args.Get(0).(result.Result[bool])
}

// DeleteAllInTimeLimit mocks the DeleteAllInTimeLimit method
func (m *MockInstanaAgentClient) DeleteAllInTimeLimit(
	ctx context.Context,
	objects []client.Object,
	timeout time.Duration,
	waitTime time.Duration,
	opts ...client.DeleteOption,
) result.Result[[]client.Object] {
	args := m.Called(append([]interface{}{ctx, objects, timeout, waitTime}, mock.Anything)...)
	return args.Get(0).(result.Result[[]client.Object])
}

// Get mocks the Get method
func (m *MockInstanaAgentClient) Get(
	ctx context.Context,
	key types.NamespacedName,
	obj client.Object,
	opts ...client.GetOption,
) error {
	args := m.Called(append([]interface{}{ctx, key, obj}, mock.Anything)...)

	// Handle the case where a function is returned
	if fn, ok := args.Get(0).(func(context.Context, types.NamespacedName, client.Object, ...client.GetOption) error); ok {
		return fn(ctx, key, obj, opts...)
	}

	return args.Error(0)
}

// GetAsResult mocks the GetAsResult method
func (m *MockInstanaAgentClient) GetAsResult(
	ctx context.Context,
	key client.ObjectKey,
	obj client.Object,
	opts ...client.GetOption,
) result.Result[client.Object] {
	args := m.Called(append([]interface{}{ctx, key, obj}, mock.Anything)...)
	return args.Get(0).(result.Result[client.Object])
}

// Status mocks the Status method
func (m *MockInstanaAgentClient) Status() client.SubResourceWriter {
	args := m.Called()
	return args.Get(0).(client.SubResourceWriter)
}

// Patch mocks the Patch method
func (m *MockInstanaAgentClient) Patch(
	ctx context.Context,
	obj client.Object,
	patch client.Patch,
	opts ...client.PatchOption,
) error {
	args := m.Called(append([]interface{}{ctx, obj, patch}, mock.Anything)...)
	return args.Error(0)
}

// Delete mocks the Delete method
func (m *MockInstanaAgentClient) Delete(
	ctx context.Context,
	obj client.Object,
	opts ...client.DeleteOption,
) error {
	args := m.Called(append([]interface{}{ctx, obj}, mock.Anything)...)
	return args.Error(0)
}

// GetNamespacesWithLabels mocks the GetNamespacesWithLabels method
func (m *MockInstanaAgentClient) GetNamespacesWithLabels(
	ctx context.Context,
) (namespaces.NamespacesDetails, error) {
	args := m.Called(ctx)
	return args.Get(0).(namespaces.NamespacesDetails), args.Error(1)
}

// Made with Bob
