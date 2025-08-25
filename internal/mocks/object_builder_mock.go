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

	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
)

// MockObjectBuilder provides a testify mock implementation of ObjectBuilder
type MockObjectBuilder struct {
	mock.Mock
}

func (m *MockObjectBuilder) Build() builder.OptionalObject {
	args := m.Called()
	return args.Get(0).(builder.OptionalObject)
}

func (m *MockObjectBuilder) ComponentName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockObjectBuilder) IsNamespaced() bool {
	args := m.Called()
	return args.Bool(0)
}
