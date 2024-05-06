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
package tls_secret

import (
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
	v1 "k8s.io/api/core/v1"
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

// ISGOMOCK indicates that this struct is a gomock mock.
func (m *MockHelpers) ISGOMOCK() struct{} {
	return struct{}{}
}

// ContainersSecretName mocks base method.
func (m *MockHelpers) ContainersSecretName() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ContainersSecretName")
	ret0, _ := ret[0].(string)
	return ret0
}

// ContainersSecretName indicates an expected call of ContainersSecretName.
func (mr *MockHelpersMockRecorder) ContainersSecretName() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ContainersSecretName", reflect.TypeOf((*MockHelpers)(nil).ContainersSecretName))
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

// ImagePullSecrets mocks base method.
func (m *MockHelpers) ImagePullSecrets() []v1.LocalObjectReference {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ImagePullSecrets")
	ret0, _ := ret[0].([]v1.LocalObjectReference)
	return ret0
}

// ImagePullSecrets indicates an expected call of ImagePullSecrets.
func (mr *MockHelpersMockRecorder) ImagePullSecrets() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ImagePullSecrets", reflect.TypeOf((*MockHelpers)(nil).ImagePullSecrets))
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

// UseContainersSecret mocks base method.
func (m *MockHelpers) UseContainersSecret() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UseContainersSecret")
	ret0, _ := ret[0].(bool)
	return ret0
}

// UseContainersSecret indicates an expected call of UseContainersSecret.
func (mr *MockHelpersMockRecorder) UseContainersSecret() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UseContainersSecret", reflect.TypeOf((*MockHelpers)(nil).UseContainersSecret))
}
