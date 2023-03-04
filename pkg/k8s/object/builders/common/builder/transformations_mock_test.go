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
package builder

import (
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
	labels "k8s.io/apimachinery/pkg/labels"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

// MockTransformations is a mock of Transformations interface.
type MockTransformations struct {
	ctrl     *gomock.Controller
	recorder *MockTransformationsMockRecorder
}

// MockTransformationsMockRecorder is the mock recorder for MockTransformations.
type MockTransformationsMockRecorder struct {
	mock *MockTransformations
}

// NewMockTransformations creates a new mock instance.
func NewMockTransformations(ctrl *gomock.Controller) *MockTransformations {
	mock := &MockTransformations{ctrl: ctrl}
	mock.recorder = &MockTransformationsMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTransformations) EXPECT() *MockTransformationsMockRecorder {
	return m.recorder
}

// ISGOMOCK indicates that this struct is a gomock mock.
func (m *MockTransformations) ISGOMOCK() struct{} {
	return struct{}{}
}

// AddCommonLabels mocks base method.
func (m *MockTransformations) AddCommonLabels(obj client.Object, component string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AddCommonLabels", obj, component)
}

// AddCommonLabels indicates an expected call of AddCommonLabels.
func (mr *MockTransformationsMockRecorder) AddCommonLabels(obj, component any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddCommonLabels", reflect.TypeOf((*MockTransformations)(nil).AddCommonLabels), obj, component)
}

// AddOwnerReference mocks base method.
func (m *MockTransformations) AddOwnerReference(obj client.Object) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AddOwnerReference", obj)
}

// AddOwnerReference indicates an expected call of AddOwnerReference.
func (mr *MockTransformationsMockRecorder) AddOwnerReference(obj any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddOwnerReference", reflect.TypeOf((*MockTransformations)(nil).AddOwnerReference), obj)
}

// PreviousGenerationsSelector mocks base method.
func (m *MockTransformations) PreviousGenerationsSelector() labels.Selector {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PreviousGenerationsSelector")
	ret0, _ := ret[0].(labels.Selector)
	return ret0
}

// PreviousGenerationsSelector indicates an expected call of PreviousGenerationsSelector.
func (mr *MockTransformationsMockRecorder) PreviousGenerationsSelector() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PreviousGenerationsSelector", reflect.TypeOf((*MockTransformations)(nil).PreviousGenerationsSelector))
}
