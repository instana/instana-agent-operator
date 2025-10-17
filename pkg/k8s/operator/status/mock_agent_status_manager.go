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

package status

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
)

// MockAgentStatusManager is a mock implementation of the AgentStatusManager interface for testing
type MockAgentStatusManager struct {
	AgentDaemonsets          []client.ObjectKey
	K8sSensorDeployment      client.ObjectKey
	AgentSecretConfig        client.ObjectKey
	AgentNamespacesConfigMap client.ObjectKey
	AgentOld                 *instanav1.InstanaAgent
}

// AddAgentDaemonset implements AgentStatusManager
func (m *MockAgentStatusManager) AddAgentDaemonset(agentDaemonset client.ObjectKey) {
	m.AgentDaemonsets = append(m.AgentDaemonsets, agentDaemonset)
}

// SetAgentOld implements AgentStatusManager
func (m *MockAgentStatusManager) SetAgentOld(agent *instanav1.InstanaAgent) {
	m.AgentOld = agent
}

// SetK8sSensorDeployment implements AgentStatusManager
func (m *MockAgentStatusManager) SetK8sSensorDeployment(k8sSensorDeployment client.ObjectKey) {
	m.K8sSensorDeployment = k8sSensorDeployment
}

// SetAgentSecretConfig implements AgentStatusManager
func (m *MockAgentStatusManager) SetAgentSecretConfig(agentSecretConfig types.NamespacedName) {
	m.AgentSecretConfig = agentSecretConfig
}

// SetAgentNamespacesConfigMap implements AgentStatusManager
func (m *MockAgentStatusManager) SetAgentNamespacesConfigMap(
	agentNamespacesConfigmap types.NamespacedName,
) {
	m.AgentNamespacesConfigMap = agentNamespacesConfigmap
}

// UpdateAgentStatus implements AgentStatusManager
func (m *MockAgentStatusManager) UpdateAgentStatus(ctx context.Context, reconcileErr error) error {
	return nil
}
