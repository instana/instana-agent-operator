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

	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/volume"
)

// MockVolumeBuilder provides a testify mock implementation of VolumeBuilder
type MockVolumeBuilder struct {
	mock.Mock
}

func (m *MockVolumeBuilder) Build(
	volumes ...volume.Volume,
) ([]corev1.Volume, []corev1.VolumeMount) {
	// Convert variadic args to []interface{} for the mock framework
	args := make([]interface{}, len(volumes))
	for i, v := range volumes {
		args[i] = v
	}
	callArgs := m.Called(args...)
	return callArgs.Get(0).([]corev1.Volume), callArgs.Get(1).([]corev1.VolumeMount)
}

func (m *MockVolumeBuilder) BuildFromUserConfig() ([]corev1.Volume, []corev1.VolumeMount) {
	args := m.Called()
	return args.Get(0).([]corev1.Volume), args.Get(1).([]corev1.VolumeMount)
}
