// Code generated by MockGen. DO NOT EDIT.
// Source: ./pkg/k8s/object/builders/common/ports/ports.go

// Package headless_service is a generated GoMock package.
package headless_service

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	v1 "github.com/instana/instana-agent-operator/api/v1"
	ports "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/ports"
	v10 "k8s.io/api/core/v1"
)

// MockPort is a mock of Port interface.
type MockPort struct {
	ctrl     *gomock.Controller
	recorder *MockPortMockRecorder
}

// MockPortMockRecorder is the mock recorder for MockPort.
type MockPortMockRecorder struct {
	mock *MockPort
}

// NewMockPort creates a new mock instance.
func NewMockPort(ctrl *gomock.Controller) *MockPort {
	mock := &MockPort{ctrl: ctrl}
	mock.recorder = &MockPortMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPort) EXPECT() *MockPortMockRecorder {
	return m.recorder
}

// String mocks base method.
func (m *MockPort) String() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "String")
	ret0, _ := ret[0].(string)
	return ret0
}

// String indicates an expected call of String.
func (mr *MockPortMockRecorder) String() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "String", reflect.TypeOf((*MockPort)(nil).String))
}

// isEnabled mocks base method.
func (m *MockPort) isEnabled(agent *v1.InstanaAgent) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "isEnabled", agent)
	ret0, _ := ret[0].(bool)
	return ret0
}

// isEnabled indicates an expected call of isEnabled.
func (mr *MockPortMockRecorder) isEnabled(agent interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "isEnabled", reflect.TypeOf((*MockPort)(nil).isEnabled), agent)
}

// portNumber mocks base method.
func (m *MockPort) portNumber() int32 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "portNumber")
	ret0, _ := ret[0].(int32)
	return ret0
}

// portNumber indicates an expected call of portNumber.
func (mr *MockPortMockRecorder) portNumber() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "portNumber", reflect.TypeOf((*MockPort)(nil).portNumber))
}

// MockPortsBuilder is a mock of PortsBuilder interface.
type MockPortsBuilder struct {
	ctrl     *gomock.Controller
	recorder *MockPortsBuilderMockRecorder
}

// MockPortsBuilderMockRecorder is the mock recorder for MockPortsBuilder.
type MockPortsBuilderMockRecorder struct {
	mock *MockPortsBuilder
}

// NewMockPortsBuilder creates a new mock instance.
func NewMockPortsBuilder(ctrl *gomock.Controller) *MockPortsBuilder {
	mock := &MockPortsBuilder{ctrl: ctrl}
	mock.recorder = &MockPortsBuilderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPortsBuilder) EXPECT() *MockPortsBuilderMockRecorder {
	return m.recorder
}

// GetContainerPorts mocks base method.
func (m *MockPortsBuilder) GetContainerPorts(ports ...ports.Port) []v10.ContainerPort {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range ports {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetContainerPorts", varargs...)
	ret0, _ := ret[0].([]v10.ContainerPort)
	return ret0
}

// GetContainerPorts indicates an expected call of GetContainerPorts.
func (mr *MockPortsBuilderMockRecorder) GetContainerPorts(ports ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetContainerPorts", reflect.TypeOf((*MockPortsBuilder)(nil).GetContainerPorts), ports...)
}

// GetServicePorts mocks base method.
func (m *MockPortsBuilder) GetServicePorts(ports ...ports.Port) []v10.ServicePort {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range ports {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetServicePorts", varargs...)
	ret0, _ := ret[0].([]v10.ServicePort)
	return ret0
}

// GetServicePorts indicates an expected call of GetServicePorts.
func (mr *MockPortsBuilderMockRecorder) GetServicePorts(ports ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetServicePorts", reflect.TypeOf((*MockPortsBuilder)(nil).GetServicePorts), ports...)
}