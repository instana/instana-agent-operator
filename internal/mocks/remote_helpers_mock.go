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
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
)

// MockRemoteHelpers provides a testify mock implementation of RemoteHelpers
type MockRemoteHelpers struct {
	mock.Mock
}

func (m *MockRemoteHelpers) ServiceAccountName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockRemoteHelpers) TLSIsEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockRemoteHelpers) TLSSecretName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockRemoteHelpers) RemoteResourcesName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockRemoteHelpers) ContainersSecretName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockRemoteHelpers) UseContainersSecret() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockRemoteHelpers) ImagePullSecrets() []corev1.LocalObjectReference {
	args := m.Called()
	return args.Get(0).([]corev1.LocalObjectReference)
}

func (m *MockRemoteHelpers) SortEnvVarsByName(envVars []corev1.EnvVar) {
	m.Called(envVars)
}
