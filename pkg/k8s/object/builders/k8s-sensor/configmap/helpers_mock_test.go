// Code generated by MockGen. DO NOT EDIT.
// Source: ./pkg/k8s/object/builders/common/helpers/helpers.go

// Package configmap is a generated GoMock package.
package configmap

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockHelpers is a mock of Helpers interface.
type MockHelpers struct {
	ctrl     *gomock.Controller
	recorder *MockHelpersMockRecorder
}

// MockHelpersMockRecorder is the mock recorder for MockHelpers.
type MockHelpersMockRecorder struct {
	mock *MockHelpers
}

// NewMockHelpers creates a new mock instance.
func NewMockHelpers(ctrl *gomock.Controller) *MockHelpers {
	mock := &MockHelpers{ctrl: ctrl}
	mock.recorder = &MockHelpersMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockHelpers) EXPECT() *MockHelpersMockRecorder {
	return m.recorder
}

// HeadlessServiceName mocks base method.
func (m *MockHelpers) HeadlessServiceName() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HeadlessServiceName")
	ret0, _ := ret[0].(string)
	return ret0
}

// HeadlessServiceName indicates an expected call of HeadlessServiceName.
func (mr *MockHelpersMockRecorder) HeadlessServiceName() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HeadlessServiceName", reflect.TypeOf((*MockHelpers)(nil).HeadlessServiceName))
}

// K8sSensorResourcesName mocks base method.
func (m *MockHelpers) K8sSensorResourcesName() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "K8sSensorResourcesName")
	ret0, _ := ret[0].(string)
	return ret0
}

// K8sSensorResourcesName indicates an expected call of K8sSensorResourcesName.
func (mr *MockHelpersMockRecorder) K8sSensorResourcesName() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "K8sSensorResourcesName", reflect.TypeOf((*MockHelpers)(nil).K8sSensorResourcesName))
}

// KeysSecretName mocks base method.
func (m *MockHelpers) KeysSecretName() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "KeysSecretName")
	ret0, _ := ret[0].(string)
	return ret0
}

// KeysSecretName indicates an expected call of KeysSecretName.
func (mr *MockHelpersMockRecorder) KeysSecretName() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "KeysSecretName", reflect.TypeOf((*MockHelpers)(nil).KeysSecretName))
}

// ServiceAccountName mocks base method.
func (m *MockHelpers) ServiceAccountName() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ServiceAccountName")
	ret0, _ := ret[0].(string)
	return ret0
}

// ServiceAccountName indicates an expected call of ServiceAccountName.
func (mr *MockHelpersMockRecorder) ServiceAccountName() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ServiceAccountName", reflect.TypeOf((*MockHelpers)(nil).ServiceAccountName))
}

// TLSIsEnabled mocks base method.
func (m *MockHelpers) TLSIsEnabled() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TLSIsEnabled")
	ret0, _ := ret[0].(bool)
	return ret0
}

// TLSIsEnabled indicates an expected call of TLSIsEnabled.
func (mr *MockHelpersMockRecorder) TLSIsEnabled() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TLSIsEnabled", reflect.TypeOf((*MockHelpers)(nil).TLSIsEnabled))
}

// TLSSecretName mocks base method.
func (m *MockHelpers) TLSSecretName() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TLSSecretName")
	ret0, _ := ret[0].(string)
	return ret0
}

// TLSSecretName indicates an expected call of TLSSecretName.
func (mr *MockHelpersMockRecorder) TLSSecretName() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TLSSecretName", reflect.TypeOf((*MockHelpers)(nil).TLSSecretName))
}
