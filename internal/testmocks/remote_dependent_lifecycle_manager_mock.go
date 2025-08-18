/*
 * (c) Copyright IBM Corp. 2025
 */

package testmocks

import (
	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MockRemoteDependentLifecycleManager is a mock implementation of RemoteDependentLifecycleManager interface
type MockRemoteDependentLifecycleManager struct {
	mock.Mock
}

// UpdateDependentLifecycleInfo mocks the UpdateDependentLifecycleInfo method
func (m *MockRemoteDependentLifecycleManager) UpdateDependentLifecycleInfo(
	currentGenerationDependents []client.Object,
) error {
	args := m.Called(currentGenerationDependents)
	return args.Error(0)
}

// CleanupDependents mocks the CleanupDependents method
func (m *MockRemoteDependentLifecycleManager) CleanupDependents(
	currentDependents ...client.Object,
) error {
	args := m.Called(currentDependents)
	return args.Error(0)
}

// Made with Bob
