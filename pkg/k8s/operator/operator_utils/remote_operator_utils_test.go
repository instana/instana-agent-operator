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
	"github.com/instana/instana-agent-operator/mocks"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	remoterbac "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/remote-agent/rbac"
	"github.com/instana/instana-agent-operator/pkg/multierror"
	"github.com/instana/instana-agent-operator/pkg/result"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestRemoteOperatorUtilsApplyAll(t *testing.T) {
	t.Run(
		"Should return an error when lifecycle.LifecycleManager.CleanupDependents returns an error", func(t *testing.T) {
			assertions := require.New(t)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Preparations and initialisations
			ctrl := gomock.NewController(t)
			instanaAgentClient := mocks.NewMockInstanaAgentClient(ctrl)
			dependentLifecycleManager := mocks.NewMockRemoteDependentLifecycleManager(ctrl)
			agent := instanav1.RemoteAgent{}

			var unstrctrd client.Object = &unstructured.Unstructured{}

			expected := errors.New("LifecycleManager cleanup failed")

			// Prepare builders
			builders := []builder.ObjectBuilder{}
			builders = append(
				builders,
				remoterbac.NewClusterRoleBindingBuilder(&agent),
				remoterbac.NewClusterRoleBindingBuilder(&agent),
			)

			// Dry-run
			instanaAgentClient.EXPECT().
				Apply(gomock.Eq(ctx), gomock.Any(), gomock.Eq(client.DryRunAll)).
				Return(result.Of(unstrctrd, nil)).
				Times(len(builders))

			// Non dry-run
			instanaAgentClient.EXPECT().
				Apply(gomock.Eq(ctx), gomock.Any(), gomock.Any()).
				Return(result.Of(unstrctrd, nil)).
				Times(len(builders))

			// Update success
			dependentLifecycleManager.EXPECT().UpdateDependentLifecycleInfo(gomock.Any()).Return(nil).AnyTimes()

			// Cleanup returns error
			dependentLifecycleManager.EXPECT().CleanupDependents(gomock.Any()).Return(expected).AnyTimes()

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
			ctrl := gomock.NewController(t)
			instanaAgentClient := mocks.NewMockInstanaAgentClient(ctrl)
			dependentLifecycleManager := mocks.NewMockRemoteDependentLifecycleManager(ctrl)
			agent := instanav1.RemoteAgent{}

			var unstrctrd client.Object = &unstructured.Unstructured{}

			expected := errors.New("LifecycleManager update failed")

			// Prepare builders
			builders := []builder.ObjectBuilder{}
			builders = append(
				builders,
				remoterbac.NewClusterRoleBuilder(&agent),
				remoterbac.NewClusterRoleBindingBuilder(&agent),
			)

			// Mock calls
			instanaAgentClient.EXPECT().
				Apply(gomock.Eq(ctx), gomock.Any(), gomock.Eq(client.DryRunAll)).
				Return(result.Of(unstrctrd, nil)).
				Times(len(builders))
			dependentLifecycleManager.EXPECT().
				UpdateDependentLifecycleInfo(gomock.Any()).
				Return(expected).
				AnyTimes()

			operatorUtils := NewRemoteOperatorUtils(ctx, instanaAgentClient, &agent, dependentLifecycleManager)

			err := operatorUtils.ApplyAll(builders...)
			assertions.Equal(expected.Error(), err.Error())
		},
	)
	t.Run(
		"Should return an error when applyAll in dry run mode causes an error", func(t *testing.T) {
			assertions := require.New(t)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Preparations and initialisations
			ctrl := gomock.NewController(t)
			instanaAgentClient := mocks.NewMockInstanaAgentClient(ctrl)
			dependentLifecycleManager := mocks.NewMockRemoteDependentLifecycleManager(ctrl)
			agent := instanav1.RemoteAgent{}

			var unstrctrd client.Object = &unstructured.Unstructured{}

			// Prepare errors
			expected := errors.New("Dry run failed")
			errBuilder := multierror.NewMultiErrorBuilder()
			errBuilder.Add(expected, expected)

			// Prepare builders
			builders := []builder.ObjectBuilder{}
			builders = append(
				builders,
				remoterbac.NewClusterRoleBuilder(&agent),
				remoterbac.NewClusterRoleBindingBuilder(&agent),
			)

			// Mock calls
			instanaAgentClient.EXPECT().
				Apply(gomock.Eq(ctx), gomock.Any(), gomock.Eq(client.DryRunAll)).
				Return(result.Of(unstrctrd, expected)).
				Times(len(builders))
			dependentLifecycleManager.EXPECT().
				UpdateDependentLifecycleInfo(gomock.Any()).
				Return(nil).
				AnyTimes()
			dependentLifecycleManager.EXPECT().
				CleanupDependents(gomock.Any()).
				Return(nil).
				AnyTimes()

			operatorUtils := NewRemoteOperatorUtils(ctx, instanaAgentClient, &agent, dependentLifecycleManager)

			err := operatorUtils.ApplyAll(builders...)
			assertions.Equal(errBuilder.Build().Error(), err.Error())
		},
	)
	t.Run(
		"Should return an error when applyAll in causes an error", func(t *testing.T) {
			assertions := require.New(t)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Preparations and initialisations
			ctrl := gomock.NewController(t)
			instanaAgentClient := mocks.NewMockInstanaAgentClient(ctrl)
			dependentLifecycleManager := mocks.NewMockRemoteDependentLifecycleManager(ctrl)
			agent := instanav1.RemoteAgent{}
			operatorUtils := NewRemoteOperatorUtils(ctx, instanaAgentClient, &agent, dependentLifecycleManager)

			var unstrctrd client.Object = &unstructured.Unstructured{}

			// Prepare errors
			expected := errors.New("Non-Dry run failed")
			errBuilder := multierror.NewMultiErrorBuilder()
			errBuilder.Add(expected, expected)

			// Prepare builders
			builders := []builder.ObjectBuilder{}
			builders = append(
				builders,
				remoterbac.NewClusterRoleBuilder(&agent),
				remoterbac.NewClusterRoleBindingBuilder(&agent),
			)

			// Mock calls
			instanaAgentClient.EXPECT().
				Apply(gomock.Eq(ctx), gomock.Any(), gomock.Eq(client.DryRunAll)).
				Return(result.Of(unstrctrd, nil)).
				Times(len(builders))
			instanaAgentClient.EXPECT().
				Apply(gomock.Eq(ctx), gomock.Any(), gomock.Any()).
				Return(result.Of(unstrctrd, expected)).
				Times(len(builders))
			dependentLifecycleManager.EXPECT().
				UpdateDependentLifecycleInfo(gomock.Any()).
				Return(nil).
				AnyTimes()

			err := operatorUtils.ApplyAll(builders...)
			assertions.Equal(errBuilder.Build().Error(), err.Error())
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
			ctrl := gomock.NewController(t)
			instanaAgentClient := mocks.NewMockInstanaAgentClient(ctrl)
			dependentLifecycleManager := mocks.NewMockRemoteDependentLifecycleManager(ctrl)
			operatorUtils := NewRemoteOperatorUtils(ctx, instanaAgentClient, &instanav1.RemoteAgent{}, dependentLifecycleManager)

			// Mock calls
			dependentLifecycleManager.EXPECT().
				CleanupDependents().
				Return(nil).
				AnyTimes()

			err := operatorUtils.DeleteAll()
			assertions.Nil(err)
		},
	)
}
