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
package status

import (
	"errors"
	"testing"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Assisted by WCA for GP
// Latest GenAI contribution: granite-20B-code-instruct-v2 model
// TestSetAgentOld checks if the SetAgentOld method correctly sets the agentOld field of the agentStatusManager struct.
func TestStatus_SetAgentOld(t *testing.T) {
	assertions := require.New(t)
	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name: "some-agent-name",
		},
	}
	manager := &agentStatusManager{}
	manager.SetAgentOld(agent)
	assertions.Equal(agent, manager.agentOld)
}

// Assisted by WCA for GP
// Latest GenAI contribution: granite-20B-code-instruct-v2 model
// TestSetK8sSensorDeployment checks if the SetK8sSensorDeployment method correctly sets the k8sSensorDeployment field of the agentStatusManager struct.
func TestStatus_SetK8sSensorDeployment(t *testing.T) {
	assertions := require.New(t)
	k8sSensorDeployment := client.ObjectKey{
		Namespace: "some-namespace",
		Name:      "some-name",
	}

	manager := &agentStatusManager{}
	manager.SetK8sSensorDeployment(k8sSensorDeployment)
	assertions.Equal(k8sSensorDeployment, manager.k8sSensorDeployment)
}

// Assisted by WCA for GP
// Latest GenAI contribution: granite-20B-code-instruct-v2 model
// TestSetAgentConfigMap checks if the SetAgentConfigMap method correctly sets the agentConfigMap field of the agentStatusManager struct.
func TestStatus_SetAgentConfigMap(t *testing.T) {
	assertions := require.New(t)
	agentConfigMap := client.ObjectKey{
		Namespace: "some-namepsace",
		Name:      "some-name",
	}

	manager := &agentStatusManager{}
	manager.SetAgentConfigMap(agentConfigMap)
	assertions.Equal(agentConfigMap, manager.agentConfigMap)
}

func TestStatus_getAgentPhase(t *testing.T) {
	for _, test := range []struct {
		name          string
		reconcileErr  error
		expectedPhase instanav1.AgentOperatorState
	}{
		{
			name:          "no_error",
			reconcileErr:  nil,
			expectedPhase: instanav1.OperatorStateRunning,
		},
		{
			name:          "error",
			reconcileErr:  errors.New("some reconcile err"),
			expectedPhase: instanav1.OperatorStateFailed,
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)

				actualPhase := getAgentPhase(test.reconcileErr)
				assertions.Equal(test.expectedPhase, actualPhase)
			})

	}
}

func TestStatus_AddAgentDaemonset(t *testing.T) {
	for _, test := range []struct {
		name             string
		initial          []client.ObjectKey
		new              client.ObjectKey
		expectedDsLength int
	}{
		{
			name:    "add_new_ds",
			initial: []client.ObjectKey{},
			new: client.ObjectKey{
				Namespace: "some-ns",
				Name:      "new-ds",
			},
			expectedDsLength: 1,
		},
		{
			name: "add_same_ds",
			initial: []client.ObjectKey{
				{
					Namespace: "some-ns",
					Name:      "initial-ds",
				},
			},
			new: client.ObjectKey{
				Namespace: "some-ns",
				Name:      "initial-ds",
			},
			expectedDsLength: 1,
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)

				manager := &agentStatusManager{
					agentDaemonsets: test.initial,
				}
				manager.AddAgentDaemonset(test.new)
				assertions.Equal(test.expectedDsLength, len(manager.agentDaemonsets))
			})

	}
}

func TestStatus_getReason(t *testing.T) {
	for _, test := range []struct {
		name           string
		err            error
		expectedReason string
	}{
		{
			name:           "null_err",
			err:            nil,
			expectedReason: "",
		},
		{
			name:           "reconcile_err",
			err:            errors.New("some reconcile err"),
			expectedReason: "some reconcile err",
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)

				assertions.Equal(test.expectedReason, getReason(test.err))
			})

	}
}

func TestStatus_toResourceInfo(t *testing.T) {
	for _, test := range []struct {
		name                 string
		obj                  appsv1.DaemonSet
		expectedResourceInfo instanav1.ResourceInfo
	}{
		{
			name: "empty_object",
			obj:  appsv1.DaemonSet{},
			expectedResourceInfo: instanav1.ResourceInfo{
				UID:  "",
				Name: "",
			},
		},
		{
			name: "non_empty_object",
			obj: appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: "some-name",
					UID:  types.UID("123"),
				},
			},
			expectedResourceInfo: instanav1.ResourceInfo{
				Name: "some-name",
				UID:  "123",
			},
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)
				result := toResourceInfo(&test.obj)

				resourceInfo, _ := result.Get()

				assertions.Equal(test.expectedResourceInfo, resourceInfo)
			})

	}
}
