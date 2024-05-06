/*
(c) Copyright IBM Corp. 2024
(c) Copyright Instana Inc.

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

// Code generated by MockGen. DO NOT EDIT.
// Source: ./pkg/k8s/object/builders/common/ports/ports.go
//
// Generated by this command:
//
//	mockgen --source ./pkg/k8s/object/builders/common/ports/ports.go --destination ./pkg/k8s/object/builders/agent/daemonset/ports_mock_test.go --package daemonset
//

// Package daemonset is a generated GoMock package.
package daemonset

import (
	reflect "reflect"

	helpers "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	ports "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/ports"
	gomock "go.uber.org/mock/gomock"
	v1 "k8s.io/api/core/v1"
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
func (m *MockPort) isEnabled(openTelemetrySettings helpers.OpenTelemetrySettings) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "isEnabled", openTelemetrySettings)
	ret0, _ := ret[0].(bool)
	return ret0
}

// isEnabled indicates an expected call of isEnabled.
func (mr *MockPortMockRecorder) isEnabled(openTelemetrySettings any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "isEnabled", reflect.TypeOf((*MockPort)(nil).isEnabled), openTelemetrySettings)
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
func (m *MockPortsBuilder) GetContainerPorts(ports ...ports.Port) []v1.ContainerPort {
	m.ctrl.T.Helper()
	varargs := []any{}
	for _, a := range ports {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetContainerPorts", varargs...)
	ret0, _ := ret[0].([]v1.ContainerPort)
	return ret0
}

// GetContainerPorts indicates an expected call of GetContainerPorts.
func (mr *MockPortsBuilderMockRecorder) GetContainerPorts(ports ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetContainerPorts", reflect.TypeOf((*MockPortsBuilder)(nil).GetContainerPorts), ports...)
}

// GetServicePorts mocks base method.
func (m *MockPortsBuilder) GetServicePorts(ports ...ports.Port) []v1.ServicePort {
	m.ctrl.T.Helper()
	varargs := []any{}
	for _, a := range ports {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetServicePorts", varargs...)
	ret0, _ := ret[0].([]v1.ServicePort)
	return ret0
}

// GetServicePorts indicates an expected call of GetServicePorts.
func (mr *MockPortsBuilderMockRecorder) GetServicePorts(ports ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetServicePorts", reflect.TypeOf((*MockPortsBuilder)(nil).GetServicePorts), ports...)
}
