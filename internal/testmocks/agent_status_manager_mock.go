/*
 * (c) Copyright IBM Corp. 2025
 */

package testmocks

import (
	"context"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MockAgentStatusManager is a mock implementation of AgentStatusManager interface
type MockAgentStatusManager struct {
	mock.Mock
}

// UpdateAgentStatus mocks the UpdateAgentStatus method
func (m *MockAgentStatusManager) UpdateAgentStatus(
	ctx context.Context,
	agent *instanav1.InstanaAgent,
) error {
	args := m.Called(ctx, agent)
	return args.Error(0)
}

// UpdateAgentRemoteStatus mocks the UpdateAgentRemoteStatus method
func (m *MockAgentStatusManager) UpdateAgentRemoteStatus(
	ctx context.Context,
	agent *instanav1.InstanaAgentRemote,
) error {
	args := m.Called(ctx, agent)
	return args.Error(0)
}

// SetAgentNamespacesConfigMap mocks the SetAgentNamespacesConfigMap method
func (m *MockAgentStatusManager) SetAgentNamespacesConfigMap(
	agentNamespacesConfigmap client.ObjectKey,
) {
	m.Called(agentNamespacesConfigmap)
}

// Made with Bob
