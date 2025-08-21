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

	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/env"
)

// MockEnvBuilder provides a testify mock implementation of EnvBuilder
type MockEnvBuilder struct {
	mock.Mock
}

func (m *MockEnvBuilder) Build(envVars ...env.EnvVar) []corev1.EnvVar {
	// Convert variadic args to []interface{} for the mock framework
	args := make([]interface{}, len(envVars))
	for i, v := range envVars {
		args[i] = v
	}
	callArgs := m.Called(args...)
	return callArgs.Get(0).([]corev1.EnvVar)
}
