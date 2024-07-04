// /*
// (c) Copyright IBM Corp. 2024
// (c) Copyright Instana Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */
//

// Code generated by MockGen. DO NOT EDIT.
package configmap

import (
	context "context"
	reflect "reflect"

	v1 "github.com/instana/instana-agent-operator/api/v1"
	gomock "go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/types"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

// MockAgentStatusManager is a mock of AgentStatusManager interface.
type MockAgentStatusManager struct {
	ctrl     *gomock.Controller
	recorder *MockAgentStatusManagerMockRecorder
}

// MockAgentStatusManagerMockRecorder is the mock recorder for MockAgentStatusManager.
type MockAgentStatusManagerMockRecorder struct {
	mock *MockAgentStatusManager
}

// NewMockAgentStatusManager creates a new mock instance.
func NewMockAgentStatusManager(ctrl *gomock.Controller) *MockAgentStatusManager {
	mock := &MockAgentStatusManager{ctrl: ctrl}
	mock.recorder = &MockAgentStatusManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAgentStatusManager) EXPECT() *MockAgentStatusManagerMockRecorder {
	return m.recorder
}

// ISGOMOCK indicates that this struct is a gomock mock.
func (m *MockAgentStatusManager) ISGOMOCK() struct{} {
	return struct{}{}
}

// AddAgentDaemonset mocks base method.
func (m *MockAgentStatusManager) AddAgentDaemonset(agentDaemonset client.ObjectKey) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AddAgentDaemonset", agentDaemonset)
}

// AddAgentDaemonset indicates an expected call of AddAgentDaemonset.
func (mr *MockAgentStatusManagerMockRecorder) AddAgentDaemonset(agentDaemonset any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddAgentDaemonset", reflect.TypeOf((*MockAgentStatusManager)(nil).AddAgentDaemonset), agentDaemonset)
}

// SetAgentConfigMap mocks base method.
func (m *MockAgentStatusManager) SetAgentConfigMap(agentConfigMap client.ObjectKey) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetAgentConfigMap", agentConfigMap)
}

// SetAgentConfigMap indicates an expected call of SetAgentConfigMap.
func (mr *MockAgentStatusManagerMockRecorder) SetAgentConfigMap(agentConfigMap any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetAgentConfigMap", reflect.TypeOf((*MockAgentStatusManager)(nil).SetAgentConfigMap), agentConfigMap)
}

// SetAgentConfigSecret implements status.AgentStatusManager.
func (m *MockAgentStatusManager) SetAgentConfigSecret(agentConfigSecret types.NamespacedName) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetAgentConfigSecret", agentConfigSecret)
}

func (mr *MockAgentStatusManagerMockRecorder) SetAgentConfigSecret(agentConfigSecret any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetAgentConfigSecret", reflect.TypeOf((*MockAgentStatusManager)(nil).SetAgentConfigSecret), agentConfigSecret)
}

// SetAgentOld mocks base method.
func (m *MockAgentStatusManager) SetAgentOld(agent *v1.InstanaAgent) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetAgentOld", agent)
}

// SetAgentOld indicates an expected call of SetAgentOld.
func (mr *MockAgentStatusManagerMockRecorder) SetAgentOld(agent any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetAgentOld", reflect.TypeOf((*MockAgentStatusManager)(nil).SetAgentOld), agent)
}

// SetK8sSensorDeployment mocks base method.
func (m *MockAgentStatusManager) SetK8sSensorDeployment(k8sSensorDeployment client.ObjectKey) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetK8sSensorDeployment", k8sSensorDeployment)
}

// SetK8sSensorDeployment indicates an expected call of SetK8sSensorDeployment.
func (mr *MockAgentStatusManagerMockRecorder) SetK8sSensorDeployment(k8sSensorDeployment any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetK8sSensorDeployment", reflect.TypeOf((*MockAgentStatusManager)(nil).SetK8sSensorDeployment), k8sSensorDeployment)
}

// UpdateAgentStatus mocks base method.
func (m *MockAgentStatusManager) UpdateAgentStatus(ctx context.Context, reconcileErr error) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateAgentStatus", ctx, reconcileErr)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateAgentStatus indicates an expected call of UpdateAgentStatus.
func (mr *MockAgentStatusManagerMockRecorder) UpdateAgentStatus(ctx, reconcileErr any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateAgentStatus", reflect.TypeOf((*MockAgentStatusManager)(nil).UpdateAgentStatus), ctx, reconcileErr)
}
