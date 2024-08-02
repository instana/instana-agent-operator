/*
(c) Copyright IBM Corp. 2024
*/

package status

import (
	"context"
	"os"
	"testing"

	"github.com/go-errors/errors"
	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/mocks"

	"github.com/instana/instana-agent-operator/pkg/result"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestUpdateAgentStatusReturnsErrorOnPatchFailure(t *testing.T) {
	assertions := require.New(t)

	ctrl := gomock.NewController(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	writer := mocks.NewMockSubResourceWriter(ctrl)
	writer.EXPECT().
		Patch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(errors.New("FAILURE")).
		AnyTimes()

	instanaAgentClient := mocks.NewMockInstanaAgentClient(ctrl)
	instanaAgentClient.EXPECT().
		GetAsResult(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(result.Of[k8sclient.Object](&unstructured.Unstructured{}, nil)).
		AnyTimes()
	instanaAgentClient.EXPECT().
		Status().
		Return(writer).
		AnyTimes()

	agentStatusManager := NewAgentStatusManager(instanaAgentClient, record.NewFakeRecorder(10))
	agentStatusManager.SetAgentOld(&instanav1.InstanaAgent{})
	agentStatusManager.SetAgentSecretConfig(types.NamespacedName{
		Name:      "SetAgentSecretConfigName",
		Namespace: "SetAgentSecretConfigNamespace",
	})
	agentStatusManager.AddAgentDaemonset(types.NamespacedName{
		Name:      "AddAgentDaemonsetName",
		Namespace: "AddAgentDaemonsetNamespace",
	})
	err := agentStatusManager.UpdateAgentStatus(ctx, nil)
	assertions.NotNil(err)
}

func TestUpdateAgentStatus(t *testing.T) {
	instanaAgent := instanav1.InstanaAgent{}
	configSecret := types.NamespacedName{
		Name:      "SetAgentSecretConfigName",
		Namespace: "SetAgentSecretConfigNamespace",
	}
	daemonsets := []*types.NamespacedName{{
		Name:      "AddAgentDaemonsetName",
		Namespace: "AddAgentDaemonsetNamespace",
	}}
	k8sSensorDeployment := &types.NamespacedName{Name: "test_name_deployment", Namespace: "test_namespace_deployment"}

	num := int64(1)
	semVer := instanav1.SemanticVersion{}
	brokenOperatorVersion := "not_a_real_version_number"

	for _, test := range []struct {
		name                  string
		getAsResultErrors     []error
		agent                 *instanav1.InstanaAgent
		configSecret          *types.NamespacedName
		daemonsets            []*types.NamespacedName
		k8sSensorDeployment   *types.NamespacedName
		reconciliationErrors  error
		expected              string
		envVarOperatorVersion *string
	}{
		{
			name:                 "Should not return errors with full configuration",
			getAsResultErrors:    []error{nil, nil, nil, nil},
			agent:                &instanaAgent,
			configSecret:         &configSecret,
			daemonsets:           daemonsets,
			k8sSensorDeployment:  k8sSensorDeployment,
			reconciliationErrors: nil,
			expected:             "",
		},
		{
			name:              "AgentStatusManager.updateWasPerformed observed-generation-does-not-match-generation",
			getAsResultErrors: []error{nil, nil, nil, nil},
			agent: &instanav1.InstanaAgent{
				Status: instanav1.InstanaAgentStatus{
					ObservedGeneration: &num,
				},
			},
			configSecret:         &configSecret,
			daemonsets:           daemonsets,
			k8sSensorDeployment:  k8sSensorDeployment,
			reconciliationErrors: nil,
			expected:             "",
		},
		{
			name:              "AgentStatusManager.updateWasPerformed operator-versions-do-not-match",
			getAsResultErrors: []error{nil, nil, nil, nil},
			agent: &instanav1.InstanaAgent{
				Status: instanav1.InstanaAgentStatus{
					ObservedGeneration: &num,
					OperatorVersion:    &semVer,
				},
				ObjectMeta: metav1.ObjectMeta{
					Generation: num,
				},
			},
			configSecret:         &configSecret,
			daemonsets:           daemonsets,
			k8sSensorDeployment:  k8sSensorDeployment,
			reconciliationErrors: nil,
			expected:             "",
		},
		{
			name:              "AgentStatusManager.updateWasPerformed operator-version-is-nil",
			getAsResultErrors: []error{nil, nil, nil, nil},
			agent: &instanav1.InstanaAgent{
				Status: instanav1.InstanaAgentStatus{
					ObservedGeneration: &num,
					OperatorVersion:    nil,
				},
				ObjectMeta: metav1.ObjectMeta{
					Generation: num,
				},
			},
			configSecret:         &configSecret,
			daemonsets:           daemonsets,
			k8sSensorDeployment:  k8sSensorDeployment,
			reconciliationErrors: nil,
			expected:             "",
		},
		{
			name:              "AgentStatusManager.updateWasPerformed broken-operator-version-environment-variable",
			getAsResultErrors: []error{nil, nil, nil, nil},
			agent: &instanav1.InstanaAgent{
				Status: instanav1.InstanaAgentStatus{
					ObservedGeneration: &num,
					OperatorVersion:    &semVer,
				},
				ObjectMeta: metav1.ObjectMeta{
					Generation: num,
				},
			},
			configSecret:          &configSecret,
			daemonsets:            daemonsets,
			k8sSensorDeployment:   k8sSensorDeployment,
			reconciliationErrors:  nil,
			expected:              "",
			envVarOperatorVersion: &brokenOperatorVersion,
		},
		{
			name:                 "Return empty when InstanaAgent is nil",
			getAsResultErrors:    []error{},
			agent:                nil,
			configSecret:         &configSecret,
			daemonsets:           daemonsets,
			k8sSensorDeployment:  k8sSensorDeployment,
			reconciliationErrors: nil,
			expected:             "",
		},
		{
			name:                 "Return empty when ConfigMap is nil",
			getAsResultErrors:    []error{nil, nil, nil, nil},
			agent:                &instanaAgent,
			configSecret:         nil,
			daemonsets:           daemonsets,
			k8sSensorDeployment:  k8sSensorDeployment,
			reconciliationErrors: nil,
			expected:             "",
		},
		{
			name:                 "Return empty when DaemonSets are nil",
			getAsResultErrors:    []error{nil, nil},
			agent:                &instanaAgent,
			configSecret:         &configSecret,
			daemonsets:           nil,
			k8sSensorDeployment:  k8sSensorDeployment,
			reconciliationErrors: nil,
			expected:             "",
		},
		{
			name:                 "Return empty when K8SSensor Deployment is nil",
			getAsResultErrors:    []error{nil, nil, nil, nil},
			agent:                &instanaAgent,
			configSecret:         &configSecret,
			daemonsets:           daemonsets,
			k8sSensorDeployment:  nil,
			reconciliationErrors: nil,
			expected:             "",
		},
		{
			name:                 "InstanaAgentClient.GetAsResult returns-error-#1",
			getAsResultErrors:    []error{errors.New("first_call_errors"), nil, nil, nil},
			agent:                &instanaAgent,
			configSecret:         &configSecret,
			daemonsets:           daemonsets,
			k8sSensorDeployment:  k8sSensorDeployment,
			reconciliationErrors: nil,
			expected:             errors.New("first_call_errors").Error(),
		},
		{
			name:                 "InstanaAgentClient.GetAsResult returns-error-#2",
			getAsResultErrors:    []error{nil, errors.New("second_call_errors"), nil, nil},
			agent:                &instanaAgent,
			configSecret:         &configSecret,
			daemonsets:           daemonsets,
			k8sSensorDeployment:  k8sSensorDeployment,
			reconciliationErrors: nil,
			expected:             errors.New("second_call_errors").Error(),
		},
		{
			name:                 "InstanaAgentClient.GetAsResult returns-error-#3",
			getAsResultErrors:    []error{nil, nil, errors.New("third_call_errors"), nil},
			agent:                &instanaAgent,
			configSecret:         &configSecret,
			daemonsets:           daemonsets,
			k8sSensorDeployment:  k8sSensorDeployment,
			reconciliationErrors: nil,
			expected:             errors.New("third_call_errors").Error(),
		},
		{
			name:                 "InstanaAgentClient.GetAsResult returns-error-#4",
			getAsResultErrors:    []error{nil, nil, nil, errors.New("fourth_call_errors")},
			agent:                &instanaAgent,
			configSecret:         &configSecret,
			daemonsets:           daemonsets,
			k8sSensorDeployment:  k8sSensorDeployment,
			reconciliationErrors: nil,
			expected:             errors.New("fourth_call_errors").Error(),
		},
		{
			name:                 "Reconciliation error does not affect returning errors",
			getAsResultErrors:    []error{nil, nil, nil, nil},
			agent:                &instanaAgent,
			configSecret:         &configSecret,
			daemonsets:           daemonsets,
			k8sSensorDeployment:  k8sSensorDeployment,
			reconciliationErrors: errors.New("reconciliation_error"),
			expected:             "",
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				if test.envVarOperatorVersion != nil {
					os.Setenv("OPERATOR_VERSION", *test.envVarOperatorVersion)
					defer func() {
						os.Setenv("OPERATOR_VERSION", "")
					}()
				}

				assertions := require.New(t)
				ctrl := gomock.NewController(t)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				writer := mocks.NewMockSubResourceWriter(ctrl)
				writer.EXPECT().
					Patch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					AnyTimes()

				instanaAgentClient := mocks.NewMockInstanaAgentClient(ctrl)
				for _, val := range test.getAsResultErrors {
					instanaAgentClient.EXPECT().
						GetAsResult(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(result.Of[k8sclient.Object](&unstructured.Unstructured{}, val)).
						Times(1)
				}
				instanaAgentClient.EXPECT().
					Status().
					Return(writer).
					AnyTimes()

				agentStatusManager := NewAgentStatusManager(instanaAgentClient, record.NewFakeRecorder(10))

				if test.agent != nil {
					agentStatusManager.SetAgentOld(test.agent)
				}
				if test.configSecret != nil {
					agentStatusManager.SetAgentSecretConfig(*test.configSecret)
				}
				if test.k8sSensorDeployment != nil {
					agentStatusManager.SetK8sSensorDeployment(*test.k8sSensorDeployment)
				}
				if len(test.daemonsets) != 0 {
					for _, val := range test.daemonsets {
						agentStatusManager.AddAgentDaemonset(*val)
					}
				}

				err := agentStatusManager.UpdateAgentStatus(ctx, test.reconciliationErrors)

				if test.expected != "" {
					assertions.Equal(test.expected, err.Error())
				} else {
					assertions.Nil(err)
				}

			},
		)
	}
}
