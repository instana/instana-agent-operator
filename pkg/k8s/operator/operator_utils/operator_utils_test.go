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

package operator_utils

import (
	"errors"
	"testing"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/mocks"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	k8ssensorrbac "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/k8s-sensor/rbac"
	"github.com/instana/instana-agent-operator/pkg/multierror"
	"github.com/instana/instana-agent-operator/pkg/pointer"
	"github.com/instana/instana-agent-operator/pkg/result"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestOperatorUtilsClusterIsOpenShift(t *testing.T) {
	for _, test := range []struct {
		name           string
		crdExists      bool
		crdExistsErr   error
		openShiftField *bool
		expected       bool
	}{
		{
			name:           "Should return true when CRD has been specified and OpenShift field is missing",
			crdExists:      true,
			crdExistsErr:   nil,
			openShiftField: nil,
			expected:       true,
		},
		{
			name:           "Should return false when neither CRD or OpenShift field have been specified",
			crdExists:      false,
			crdExistsErr:   nil,
			openShiftField: nil,
			expected:       false,
		},
		{
			name:           "Should get an error correctly by AgentClient.Exists",
			crdExists:      false,
			crdExistsErr:   errors.New("qwerty"),
			openShiftField: nil,
			expected:       false,
		},
		{
			name:           "Should return true when user has specified OpenShift in the spec as true",
			crdExists:      false,
			crdExistsErr:   nil,
			openShiftField: pointer.To(true),
			expected:       true,
		},
		{
			name:           "Should return false when user has specified OpenShift in the spec as false",
			crdExists:      false,
			crdExistsErr:   nil,
			openShiftField: pointer.To(true),
			expected:       true,
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				ctrl := gomock.NewController(t)
				instanaAgentClient := mocks.NewMockInstanaAgentClient(ctrl)
				dependentLifecycleManager := mocks.NewMockDependentLifecycleManager(ctrl)

				gvk := schema.GroupVersionKind{
					Group:   "apiextensions.k8s.io",
					Version: "v1",
					Kind:    "CustomResourceDefinition",
				}
				key := types.NamespacedName{
					Name: "clusteroperators.config.openshift.io",
				}

				instanaAgentClient.EXPECT().
					Exists(gomock.Eq(ctx), gomock.Eq(gvk), gomock.Eq(key)).
					Return(result.Of(test.crdExists, test.crdExistsErr)).
					AnyTimes()

				instanaAgent := instanav1.InstanaAgent{}
				instanaAgent.Spec.OpenShift = test.openShiftField

				operatorUtils := NewOperatorUtils(ctx, instanaAgentClient, &instanaAgent, dependentLifecycleManager)
				isOpenShift, err := operatorUtils.ClusterIsOpenShift()

				assertions.Equal(test.expected, isOpenShift)
				assertions.Equal(test.crdExistsErr, err)
			},
		)
	}
}

func TestOperatorUtilsApplyAll(t *testing.T) {
	t.Run(
		"Should return an error when lifecycle.LifecycleManager.CleanupDependents returns an error", func(t *testing.T) {
			assertions := require.New(t)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Preparations and initialisations
			ctrl := gomock.NewController(t)
			instanaAgentClient := mocks.NewMockInstanaAgentClient(ctrl)
			dependentLifecycleManager := mocks.NewMockDependentLifecycleManager(ctrl)
			agent := instanav1.InstanaAgent{}

			var unstrctrd client.Object = &unstructured.Unstructured{}

			expected := errors.New("LifecycleManager cleanup failed")

			// Prepare builders
			builders := []builder.ObjectBuilder{}
			builders = append(
				builders,
				k8ssensorrbac.NewClusterRoleBuilder(&agent),
				k8ssensorrbac.NewClusterRoleBindingBuilder(&agent),
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

			operatorUtils := NewOperatorUtils(ctx, instanaAgentClient, &agent, dependentLifecycleManager)

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
			dependentLifecycleManager := mocks.NewMockDependentLifecycleManager(ctrl)
			agent := instanav1.InstanaAgent{}

			var unstrctrd client.Object = &unstructured.Unstructured{}

			expected := errors.New("LifecycleManager update failed")

			// Prepare builders
			builders := []builder.ObjectBuilder{}
			builders = append(
				builders,
				k8ssensorrbac.NewClusterRoleBuilder(&agent),
				k8ssensorrbac.NewClusterRoleBindingBuilder(&agent),
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

			operatorUtils := NewOperatorUtils(ctx, instanaAgentClient, &agent, dependentLifecycleManager)

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
			dependentLifecycleManager := mocks.NewMockDependentLifecycleManager(ctrl)
			agent := instanav1.InstanaAgent{}

			var unstrctrd client.Object = &unstructured.Unstructured{}

			// Prepare errors
			expected := errors.New("Dry run failed")
			errBuilder := multierror.NewMultiErrorBuilder()
			errBuilder.Add(expected, expected)

			// Prepare builders
			builders := []builder.ObjectBuilder{}
			builders = append(
				builders,
				k8ssensorrbac.NewClusterRoleBuilder(&agent),
				k8ssensorrbac.NewClusterRoleBindingBuilder(&agent),
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

			operatorUtils := NewOperatorUtils(ctx, instanaAgentClient, &agent, dependentLifecycleManager)

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
			dependentLifecycleManager := mocks.NewMockDependentLifecycleManager(ctrl)
			agent := instanav1.InstanaAgent{}
			operatorUtils := NewOperatorUtils(ctx, instanaAgentClient, &agent, dependentLifecycleManager)

			var unstrctrd client.Object = &unstructured.Unstructured{}

			// Prepare errors
			expected := errors.New("Non-Dry run failed")
			errBuilder := multierror.NewMultiErrorBuilder()
			errBuilder.Add(expected, expected)

			// Prepare builders
			builders := []builder.ObjectBuilder{}
			builders = append(
				builders,
				k8ssensorrbac.NewClusterRoleBuilder(&agent),
				k8ssensorrbac.NewClusterRoleBindingBuilder(&agent),
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

func TestOperatorUtilsDeleteAll(t *testing.T) {
	t.Run(
		"DeleteAll calls DependentLifecycleManager.CleanupDependents", func(t *testing.T) {
			assertions := require.New(t)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Preparations and initialisations
			ctrl := gomock.NewController(t)
			instanaAgentClient := mocks.NewMockInstanaAgentClient(ctrl)
			dependentLifecycleManager := mocks.NewMockDependentLifecycleManager(ctrl)
			operatorUtils := NewOperatorUtils(ctx, instanaAgentClient, &instanav1.InstanaAgent{}, dependentLifecycleManager)

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
