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
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
)

// MockAgentStatusManager provides a testify mock implementation of AgentStatusManager
type MockAgentStatusManager struct {
	mock.Mock
}

func (m *MockAgentStatusManager) AddAgentDaemonset(agentDaemonset client.ObjectKey) {
	m.Called(agentDaemonset)
}

func (m *MockAgentStatusManager) SetAgentOld(agent *instanav1.InstanaAgent) {
	m.Called(agent)
}

func (m *MockAgentStatusManager) SetK8sSensorDeployment(k8sSensorDeployment client.ObjectKey) {
	m.Called(k8sSensorDeployment)
}

func (m *MockAgentStatusManager) SetAgentSecretConfig(agentSecretConfig client.ObjectKey) {
	m.Called(agentSecretConfig)
}

func (m *MockAgentStatusManager) SetAgentNamespacesConfigMap(
	agentNamespacesConfigmap client.ObjectKey,
) {
	m.Called(agentNamespacesConfigmap)
}

func (m *MockAgentStatusManager) UpdateAgentStatus(ctx context.Context, reconcileErr error) error {
	args := m.Called(ctx, reconcileErr)
	return args.Error(0)
}
