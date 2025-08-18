/*
 * (c) Copyright IBM Corp. 2025
 */

package testmocks

import (
	"context"

	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MockSubResourceWriter is a mock implementation of client.SubResourceWriter interface
type MockSubResourceWriter struct {
	mock.Mock
}

// Create mocks the Create method
func (m *MockSubResourceWriter) Create(
	ctx context.Context,
	obj client.Object,
	subResource client.Object,
	opts ...client.SubResourceCreateOption,
) error {
	args := m.Called(append([]interface{}{ctx, obj, subResource}, mock.Anything)...)
	return args.Error(0)
}

// Update mocks the Update method
func (m *MockSubResourceWriter) Update(
	ctx context.Context,
	obj client.Object,
	opts ...client.SubResourceUpdateOption,
) error {
	args := m.Called(append([]interface{}{ctx, obj}, mock.Anything)...)
	return args.Error(0)
}

// Patch mocks the Patch method
func (m *MockSubResourceWriter) Patch(
	ctx context.Context,
	obj client.Object,
	patch client.Patch,
	opts ...client.SubResourcePatchOption,
) error {
	args := m.Called(append([]interface{}{ctx, obj, patch}, mock.Anything)...)
	return args.Error(0)
}

// ExpectUpdate is a helper method for setting up Update expectations
func (m *MockSubResourceWriter) ExpectUpdate(ctx context.Context, obj client.Object) *mock.Call {
	return m.On("Update", ctx, obj, mock.Anything)
}

// ExpectPatch is a helper method for setting up Patch expectations
func (m *MockSubResourceWriter) ExpectPatch(
	ctx context.Context,
	obj client.Object,
	patch client.Patch,
) *mock.Call {
	return m.On("Patch", ctx, obj, patch, mock.Anything)
}

// Made with Bob
