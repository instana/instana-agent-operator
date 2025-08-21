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

package lifecycle

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"testing"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/internal/mocks"
	"github.com/instana/instana-agent-operator/pkg/multierror"
	"github.com/instana/instana-agent-operator/pkg/result"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func genMockObjs(amount int) []client.Object {
	objects := []client.Object{}
	for i := 0; i < amount; i++ {
		unstrctrd := &unstructured.Unstructured{}
		unstrctrd.SetName("i" + strconv.Itoa(i))
		unstrctrd.SetKind("agent-data")
		objects = append(objects, unstrctrd)
	}
	return objects
}

func TestAsObjectConversion(t *testing.T) {
	conversion := asObject(unstructured.Unstructured{})
	var expectedType client.Object = conversion
	assert.IsType(t, expectedType, conversion)
}

// TestClenupDependentsDeletesUnmatchedData - In an instance where ConfigMap.Data
// contains more than the current generated key to hold data in, the code will
// remove that field from the array
func TestCleanupDependentsDeletesUnmatchedData(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	instanaAgentClient := &mocks.MockInstanaAgentClient{}
	defer instanaAgentClient.AssertExpectations(t)

	// Create two client.Object arrays which have some overlap for comparisons and deletion calls
	oldDependentsJson := genMockObjs(10)
	currentDependentsJson := genMockObjs(5)
	currentDependentsJson = append(currentDependentsJson, oldDependentsJson[:5]...)

	instanaAgentClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			config := args.Get(2).(*corev1.ConfigMap)
			config.TypeMeta = metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			}
			config.ObjectMeta = metav1.ObjectMeta{
				Name:      "asdasd",
				Namespace: "asdasd",
			}

			config.Data = make(map[string]string, 1)
			olderDependentsJsonString, _ := json.Marshal(asUnstructureds(oldDependentsJson...))
			config.Data["v0.0.1-dev"] = string(olderDependentsJsonString)
			config.Data["v0.0.1-dev-to-be-deleted"] = string(olderDependentsJsonString)
		}).Return(nil)

	instanaAgentClient.On("DeleteAllInTimeLimit",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(result.Of(genMockObjs(1), nil)).
		Times(2)

	var obj client.Object = &unstructured.Unstructured{}
	instanaAgentClient.On("Apply", mock.Anything, mock.Anything, mock.Anything).
		Return(result.Of(obj, nil))

	dependentLifecycleManager := NewDependentLifecycleManager(
		ctx,
		&instanav1.InstanaAgent{},
		instanaAgentClient,
	)

	err := dependentLifecycleManager.CleanupDependents(currentDependentsJson...)
	assert.Nil(t, err)
}

// TestCleanupDependentsDeleteAllReturnsError - returns an error from the function
// delete all and returns that correctly back to the caller
func TestCleanupDependentsDeleteAllReturnsError(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	instanaAgentClient := &mocks.MockInstanaAgentClient{}
	defer instanaAgentClient.AssertExpectations(t)

	instanaAgentClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			config := args.Get(2).(*corev1.ConfigMap)
			config.TypeMeta = metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			}
			config.ObjectMeta = metav1.ObjectMeta{
				Name:      "asdasd",
				Namespace: "asdasd",
			}

			config.Data = make(map[string]string, 1)
			currentDependentsJson, _ := json.Marshal(asUnstructureds(genMockObjs(12)...))
			config.Data["v0.0.1-dev_1234"] = string(currentDependentsJson)
		}).Return(nil)

	errBuilder := multierror.NewMultiErrorBuilder()
	expected := errors.New("Error returned from d.deleteAll")
	errBuilder.Add(expected)

	instanaAgentClient.On("DeleteAllInTimeLimit",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(result.Of(genMockObjs(1), expected))

	var obj client.Object = &unstructured.Unstructured{}
	instanaAgentClient.On("Apply", mock.Anything, mock.Anything, mock.Anything).
		Return(result.Of(obj, nil))

	dependentLifecycleManager := NewDependentLifecycleManager(
		ctx,
		&instanav1.InstanaAgent{},
		instanaAgentClient,
	)

	err := dependentLifecycleManager.CleanupDependents(genMockObjs(10)...)

	assert.True(t, errors.Is(errBuilder.Build(), err))
}

func TestCleanupDependents(t *testing.T) {
	for _, test := range []struct {
		name             string
		expected         string
		instanaAgent     instanav1.InstanaAgent
		generatedObjects int
		clientGetterFunc func(
			_ context.Context,
			_ types.NamespacedName,
			config *corev1.ConfigMap,
		) error
	}{
		{
			name:             "Completely successsful run of CleanupDependents without ConfigMap.Data containing any data",
			expected:         "",
			generatedObjects: 0,
			clientGetterFunc: func(
				_ context.Context,
				_ types.NamespacedName,
				config *corev1.ConfigMap,
			) error {
				config.TypeMeta = metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				}
				config.ObjectMeta = metav1.ObjectMeta{
					Name:      "asdasd",
					Namespace: "asdasd",
				}

				config.Data = make(map[string]string, 1)
				currentDependentsJson, _ := json.Marshal(asUnstructureds(genMockObjs(0)...))
				config.Data["v0.0.1-dev_1234"] = string(currentDependentsJson)

				return nil
			},
			instanaAgent: instanav1.InstanaAgent{ObjectMeta: metav1.ObjectMeta{Generation: 1234}},
		},
		{
			name:             "Completely successful run of CleanupDependents with ConfigMap.Data containing already data",
			expected:         "",
			generatedObjects: 10,
			clientGetterFunc: func(
				_ context.Context,
				_ types.NamespacedName,
				config *corev1.ConfigMap,
			) error {
				config.TypeMeta = metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				}
				config.ObjectMeta = metav1.ObjectMeta{
					Name:      "asdasd",
					Namespace: "asdasd",
				}

				config.Data = make(map[string]string, 1)
				currentDependentsJson, _ := json.Marshal(asUnstructureds(genMockObjs(12)...))
				config.Data["v0.0.1-dev_1234"] = string(currentDependentsJson)

				return nil
			},
			instanaAgent: instanav1.InstanaAgent{ObjectMeta: metav1.ObjectMeta{Generation: 1234}},
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				instanaAgentClient := &mocks.MockInstanaAgentClient{}
				defer instanaAgentClient.AssertExpectations(t)
				getReturn := test.clientGetterFunc(
					context.Background(),
					types.NamespacedName{},
					&corev1.ConfigMap{})
				instanaAgentClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						if getReturn == nil {
							_ = test.clientGetterFunc(
								args.Get(0).(context.Context),
								args.Get(1).(types.NamespacedName),
								args.Get(2).(*corev1.ConfigMap))
						}
					}).
					Return(getReturn)
				instanaAgentClient.On("DeleteAllInTimeLimit",
					mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(result.Of(genMockObjs(test.generatedObjects), nil))
				var obj client.Object = &unstructured.Unstructured{}
				instanaAgentClient.On("Apply", mock.Anything, mock.Anything, mock.Anything).
					Return(result.Of(obj, nil))

				dependentLifecycleManager := NewDependentLifecycleManager(
					ctx,
					&test.instanaAgent,
					instanaAgentClient,
				)

				err := dependentLifecycleManager.CleanupDependents(
					genMockObjs(test.generatedObjects)...)
				assertions.Nil(err)
			},
		)
	}

}

func TestUpdateLifecycleInfo(t *testing.T) {
	for _, test := range []struct {
		name             string
		expected         string
		instanaAgent     instanav1.InstanaAgent
		clientGetterFunc func(
			_ context.Context,
			_ types.NamespacedName,
			config *corev1.ConfigMap,
		) error
	}{
		{
			name:     "Should get ConfigMap successfully and do changes accordingly UpdateLifecycleInfo with it",
			expected: "",
			clientGetterFunc: func(
				_ context.Context,
				_ types.NamespacedName,
				config *corev1.ConfigMap,
			) error {
				config.TypeMeta = metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				}
				config.ObjectMeta = metav1.ObjectMeta{
					Name:      "asdasd",
					Namespace: "asdasd",
				}
				return nil
			},
			instanaAgent: instanav1.InstanaAgent{},
		},
		{
			name:     "Should initialize ConfigMap if NotFound error was given and UpdateLifecycleInfo",
			expected: "",
			clientGetterFunc: func(
				_ context.Context,
				_ types.NamespacedName,
				config *corev1.ConfigMap,
			) error {
				return k8serrors.NewNotFound(schema.GroupResource{}, "asdasd")
			},
			instanaAgent: instanav1.InstanaAgent{},
		},
		{
			name:     "Should initialize empty ConfigMap on error and UpdateLifecycleInfo",
			expected: "",
			clientGetterFunc: func(
				_ context.Context,
				_ types.NamespacedName,
				config *corev1.ConfigMap,
			) error {
				return errors.New("An expected error occurred")
			},
			instanaAgent: instanav1.InstanaAgent{},
		},
		{
			name:     "Should run with existing ConfigMap.Data field populated and UpdateLifecycleInfo",
			expected: "",
			clientGetterFunc: func(
				_ context.Context,
				_ types.NamespacedName,
				config *corev1.ConfigMap,
			) error {
				config.TypeMeta = metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				}
				config.ObjectMeta = metav1.ObjectMeta{
					Name:      "asdasd",
					Namespace: "asdasd",
				}
				config.Data = make(map[string]string, 1)
				currentDependentsJson, _ := json.Marshal(asUnstructureds())
				config.Data["v0.0.1-dev_1234"] = string(currentDependentsJson)
				return nil
			},
			instanaAgent: instanav1.InstanaAgent{ObjectMeta: metav1.ObjectMeta{Generation: 1234}},
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				instanaAgentClient := &mocks.MockInstanaAgentClient{}
				defer instanaAgentClient.AssertExpectations(t)
				getReturn := test.clientGetterFunc(
					context.Background(),
					types.NamespacedName{},
					&corev1.ConfigMap{})
				instanaAgentClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						if getReturn == nil {
							_ = test.clientGetterFunc(
								args.Get(0).(context.Context),
								args.Get(1).(types.NamespacedName),
								args.Get(2).(*corev1.ConfigMap))
						}
					}).
					Return(getReturn)
				instanaAgentClient.On("Apply", mock.Anything, mock.Anything, mock.Anything).
					Return(result.Of[client.Object](&unstructured.Unstructured{}, nil))

				dependentLifecycleManager := NewDependentLifecycleManager(
					ctx,
					&test.instanaAgent,
					instanaAgentClient,
				)
				err := dependentLifecycleManager.UpdateDependentLifecycleInfo([]client.Object{})
				assert.Nil(t, err)
			},
		)
	}
}
