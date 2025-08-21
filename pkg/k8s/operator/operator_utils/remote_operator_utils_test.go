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

package operator_utils

import (
	"errors"
	"testing"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/internal/mocks"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestRemoteOperatorUtilsApplyAll(t *testing.T) {
	t.Run(
		"Should return an error when lifecycle.LifecycleManager.CleanupDependents returns an error", func(t *testing.T) {
			assertions := require.New(t)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Preparations and initialisations
			instanaAgentClient := &mocks.MockInstanaAgentClient{}
			defer instanaAgentClient.AssertExpectations(t)
			dependentLifecycleManager := &mocks.MockRemoteDependentLifecycleManager{}
			defer dependentLifecycleManager.AssertExpectations(t)
			agent := instanav1.InstanaAgentRemote{}

			expected := errors.New("LifecycleManager cleanup failed")

			// Prepare builders
			builders := []builder.ObjectBuilder{}

			// These Apply calls should not happen with empty builders - remove expectations
			// instanaAgentClient.On("Apply", ctx, mock.Anything, []client.PatchOption{client.DryRunAll}).
			// 	Return(result.Of(unstrctrd, nil)).
			// 	Times(len(builders))

			// instanaAgentClient.On("Apply", ctx, mock.Anything, []client.PatchOption{}).
			// 	Return(result.Of(unstrctrd, nil)).
			// 	Times(len(builders))

			// Update success
			dependentLifecycleManager.On("UpdateDependentLifecycleInfo", mock.Anything).Return(nil)

			// Cleanup returns error
			dependentLifecycleManager.On("CleanupDependents", mock.Anything).Return(expected)

			operatorUtils := NewRemoteOperatorUtils(ctx, instanaAgentClient, &agent, dependentLifecycleManager)

			err := operatorUtils.ApplyAll(builders...)
			assertions.Equal(expected.Error(), err.Error())
		},
	)
	t.Run(
		"Should return an error when lifecycle.LifecycleManager.UpdateDependentLifecycleInfo causes an error", func(t *testing.T) {
			assertions := require.New(t)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Preparations and initialisations
			instanaAgentClient := &mocks.MockInstanaAgentClient{}
			defer instanaAgentClient.AssertExpectations(t)
			dependentLifecycleManager := &mocks.MockRemoteDependentLifecycleManager{}
			defer dependentLifecycleManager.AssertExpectations(t)
			agent := instanav1.InstanaAgentRemote{}

			expected := errors.New("LifecycleManager update failed")

			// Prepare builders
			builders := []builder.ObjectBuilder{}

			// Mock calls - Apply should not be called with empty builders
			// instanaAgentClient.On("Apply", ctx, mock.Anything, []client.PatchOption{client.DryRunAll}).
			// 	Return(result.Of(unstrctrd, nil)).
			// 	Times(len(builders))
			dependentLifecycleManager.On("UpdateDependentLifecycleInfo", mock.Anything).
				Return(expected)

			operatorUtils := NewRemoteOperatorUtils(ctx, instanaAgentClient, &agent, dependentLifecycleManager)

			err := operatorUtils.ApplyAll(builders...)
			assertions.Equal(expected.Error(), err.Error())
		},
	)
}

func TestRemoteOperatorUtilsDeleteAll(t *testing.T) {
	t.Run(
		"DeleteAll calls DependentLifecycleManager.CleanupDependents", func(t *testing.T) {
			assertions := require.New(t)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Preparations and initialisations
			instanaAgentClient := &mocks.MockInstanaAgentClient{}
			defer instanaAgentClient.AssertExpectations(t)
			dependentLifecycleManager := &mocks.MockRemoteDependentLifecycleManager{}
			defer dependentLifecycleManager.AssertExpectations(t)
			operatorUtils := NewRemoteOperatorUtils(ctx, instanaAgentClient, &instanav1.InstanaAgentRemote{}, dependentLifecycleManager)

			// Mock calls
			dependentLifecycleManager.On("CleanupDependents", ([]client.Object)(nil)).Return(nil)

			err := operatorUtils.DeleteAll()
			assertions.Nil(err)
		},
	)
}
