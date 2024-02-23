// Code generated by MockGen. DO NOT EDIT.
// Source: ./pkg/k8s/operator/status/status.go

// Package configmap is a generated GoMock package.
package configmap

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	v1 "github.com/instana/instana-agent-operator/api/v1"
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

// AddAgentDaemonset mocks base method.
func (m *MockAgentStatusManager) AddAgentDaemonset(agentDaemonset client.ObjectKey) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AddAgentDaemonset", agentDaemonset)
}

// AddAgentDaemonset indicates an expected call of AddAgentDaemonset.
func (mr *MockAgentStatusManagerMockRecorder) AddAgentDaemonset(agentDaemonset interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddAgentDaemonset", reflect.TypeOf((*MockAgentStatusManager)(nil).AddAgentDaemonset), agentDaemonset)
}

// SetAgentConfigMap mocks base method.
func (m *MockAgentStatusManager) SetAgentConfigMap(agentConfigMap client.ObjectKey) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetAgentConfigMap", agentConfigMap)
}

// SetAgentConfigMap indicates an expected call of SetAgentConfigMap.
func (mr *MockAgentStatusManagerMockRecorder) SetAgentConfigMap(agentConfigMap interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetAgentConfigMap", reflect.TypeOf((*MockAgentStatusManager)(nil).SetAgentConfigMap), agentConfigMap)
}

// SetAgentOld mocks base method.
func (m *MockAgentStatusManager) SetAgentOld(agent *v1.InstanaAgent) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetAgentOld", agent)
}

// SetAgentOld indicates an expected call of SetAgentOld.
func (mr *MockAgentStatusManagerMockRecorder) SetAgentOld(agent interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetAgentOld", reflect.TypeOf((*MockAgentStatusManager)(nil).SetAgentOld), agent)
}

// SetK8sSensorDeployment mocks base method.
func (m *MockAgentStatusManager) SetK8sSensorDeployment(k8sSensorDeployment client.ObjectKey) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetK8sSensorDeployment", k8sSensorDeployment)
}

// SetK8sSensorDeployment indicates an expected call of SetK8sSensorDeployment.
func (mr *MockAgentStatusManagerMockRecorder) SetK8sSensorDeployment(k8sSensorDeployment interface{}) *gomock.Call {
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
func (mr *MockAgentStatusManagerMockRecorder) UpdateAgentStatus(ctx, reconcileErr interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateAgentStatus", reflect.TypeOf((*MockAgentStatusManager)(nil).UpdateAgentStatus), ctx, reconcileErr)
}
