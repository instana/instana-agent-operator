/*
 * (c) Copyright IBM Corp. 2025
 */

package testmocks

import (
	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MockDependentLifecycleManager is a mock implementation of DependentLifecycleManager interface
type MockDependentLifecycleManager struct {
	mock.Mock
}

// UpdateDependentLifecycleInfo mocks the UpdateDependentLifecycleInfo method
func (m *MockDependentLifecycleManager) UpdateDependentLifecycleInfo(
	currentGenerationDependents []client.Object,
) error {
	args := m.Called(currentGenerationDependents)
	return args.Error(0)
}

// CleanupDependents mocks the CleanupDependents method
func (m *MockDependentLifecycleManager) CleanupDependents(
	currentDependents ...client.Object,
) error {
	args := m.Called(currentDependents)
	return args.Error(0)
}

// Made with Bob
