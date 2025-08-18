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

package lifecycle

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/internal/testmocks"
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

func genMockObjsRemote(amount int) []client.Object {
	objects := []client.Object{}
	for i := 0; i < amount; i++ {
		unstrctrd := &unstructured.Unstructured{}
		unstrctrd.SetName("i" + strconv.Itoa(i))
		unstrctrd.SetKind("remote-agent-data")
		objects = append(objects, unstrctrd)
	}
	return objects
}

func TestAsObjectConversionRemote(t *testing.T) {
	conversion := asObject(unstructured.Unstructured{})
	var expectedType client.Object = conversion
	assert.IsType(t, expectedType, conversion)
}

// TestClenupDependentsDeletesUnmatchedData - In an instance where ConfigMap.Data
// contains more than the current generated key to hold data in, the code will
// remove that field from the array
func TestCleanupDependentsDeletesUnmatchedDataRemote(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	instanaAgentClient := new(testmocks.MockInstanaAgentClient)

	// Create two client.Object arrays which have some overlap for comparisons and deletion calls
	oldDependentsJson := genMockObjsRemote(10)
	currentDependentsJson := genMockObjsRemote(5)
	currentDependentsJson = append(currentDependentsJson, oldDependentsJson[:5]...)

	instanaAgentClient.On("Get",
		mock.Anything,                        // ctx
		mock.Anything,                        // namespacedName
		mock.AnythingOfType("*v1.ConfigMap"), // config
		mock.Anything,                        // opts
	).Run(func(args mock.Arguments) {
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
		mock.Anything, // ctx
		mock.Anything, // objects
		mock.Anything, // timeout
		mock.Anything, // retryInterval
		mock.Anything, // opts
	).Return(result.Of(genMockObjsRemote(1), nil)).Times(2)

	var obj client.Object = &unstructured.Unstructured{}
	instanaAgentClient.On("Apply",
		mock.Anything, // ctx
		mock.Anything, // obj
		mock.Anything, // opts
	).Return(result.Of(obj, nil))

	dependentLifecycleManager := NewRemoteDependentLifecycleManager(
		ctx,
		&instanav1.InstanaAgentRemote{},
		instanaAgentClient,
	)

	err := dependentLifecycleManager.CleanupDependents(currentDependentsJson...)
	assert.Nil(t, err)
	instanaAgentClient.AssertExpectations(t)
}

// TestCleanupDependentsDeleteAllReturnsError - returns an error from the function
// delete all and returns that correctly back to the caller
func TestCleanupDependentsDeleteAllReturnsErrorRemote(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	instanaAgentClient := new(testmocks.MockInstanaAgentClient)

	instanaAgentClient.On("Get",
		mock.Anything,                        // ctx
		mock.Anything,                        // namespacedName
		mock.AnythingOfType("*v1.ConfigMap"), // config
		mock.Anything,                        // opts
	).Run(func(args mock.Arguments) {
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
		currentDependentsJson, _ := json.Marshal(asUnstructureds(genMockObjsRemote(12)...))
		config.Data["v0.0.1-dev_1234"] = string(currentDependentsJson)
	}).Return(nil)

	errBuilder := multierror.NewMultiErrorBuilder()
	expected := errors.New("Error returned from d.deleteAll")
	errBuilder.Add(expected)

	instanaAgentClient.On("DeleteAllInTimeLimit",
		mock.Anything, // ctx
		mock.Anything, // objects
		mock.Anything, // timeout
		mock.Anything, // retryInterval
		mock.Anything, // opts
	).Return(result.Of(genMockObjsRemote(1), expected))

	var obj client.Object = &unstructured.Unstructured{}
	instanaAgentClient.On("Apply",
		mock.Anything, // ctx
		mock.Anything, // obj
		mock.Anything, // opts
	).Return(result.Of(obj, nil))

	dependentLifecycleManager := NewRemoteDependentLifecycleManager(
		ctx,
		&instanav1.InstanaAgentRemote{},
		instanaAgentClient,
	)

	err := dependentLifecycleManager.CleanupDependents(genMockObjsRemote(10)...)

	assert.True(t, errors.Is(errBuilder.Build(), err))
	instanaAgentClient.AssertExpectations(t)
}

func TestCleanupDependentsRemote(t *testing.T) {
	for _, test := range []struct {
		name             string
		expected         string
		instanaAgent     instanav1.InstanaAgentRemote
		generatedObjects int
		clientGetterFunc func(
			_ context.Context,
			_ types.NamespacedName,
			config *corev1.ConfigMap,
			opts ...client.GetOption,
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
				opts ...client.GetOption,
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
				currentDependentsJson, _ := json.Marshal(asUnstructureds(genMockObjsRemote(0)...))
				config.Data["v0.0.1-dev_1234"] = string(currentDependentsJson)

				return nil
			},
			instanaAgent: instanav1.InstanaAgentRemote{ObjectMeta: metav1.ObjectMeta{Generation: 1234}},
		},
		{
			name:             "Completely successful run of CleanupDependents with ConfigMap.Data containing already data",
			expected:         "",
			generatedObjects: 10,
			clientGetterFunc: func(
				_ context.Context,
				_ types.NamespacedName,
				config *corev1.ConfigMap,
				opts ...client.GetOption,
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
				currentDependentsJson, _ := json.Marshal(asUnstructureds(genMockObjsRemote(12)...))
				config.Data["v0.0.1-dev_1234"] = string(currentDependentsJson)

				return nil
			},
			instanaAgent: instanav1.InstanaAgentRemote{ObjectMeta: metav1.ObjectMeta{Generation: 1234}},
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				instanaAgentClient := new(testmocks.MockInstanaAgentClient)
				instanaAgentClient.On("Get",
					mock.Anything,                        // ctx
					mock.Anything,                        // namespacedName
					mock.AnythingOfType("*v1.ConfigMap"), // config
					mock.Anything,                        // opts
				).Run(func(args mock.Arguments) {
					config := args.Get(2).(*corev1.ConfigMap)
					err := test.clientGetterFunc(nil, types.NamespacedName{}, config)
					if err != nil {
						panic(err)
					}
				}).Return(nil)

				instanaAgentClient.On("DeleteAllInTimeLimit",
					mock.Anything, // ctx
					mock.Anything, // objects
					mock.Anything, // timeout
					mock.Anything, // retryInterval
					mock.Anything, // opts
				).Return(result.Of(genMockObjsRemote(test.generatedObjects), nil))

				var obj client.Object = &unstructured.Unstructured{}
				instanaAgentClient.On("Apply",
					mock.Anything, // ctx
					mock.Anything, // obj
					mock.Anything, // opts
				).Return(result.Of(obj, nil))

				dependentLifecycleManager := NewRemoteDependentLifecycleManager(
					ctx,
					&test.instanaAgent,
					instanaAgentClient,
				)

				err := dependentLifecycleManager.CleanupDependents(genMockObjsRemote(test.generatedObjects)...)
				assertions.Nil(err)
				instanaAgentClient.AssertExpectations(t)
			},
		)
	}
}

func TestUpdateLifecycleInfoRemote(t *testing.T) {
	for _, test := range []struct {
		name             string
		expected         string
		instanaAgent     instanav1.InstanaAgentRemote
		clientGetterFunc func(
			_ context.Context,
			_ types.NamespacedName,
			config *corev1.ConfigMap,
			opts ...client.GetOption,
		) error
	}{
		{
			name:     "Should get ConfigMap successfully and do changes accordingly UpdateLifecycleInfo with it",
			expected: "",
			clientGetterFunc: func(
				_ context.Context,
				_ types.NamespacedName,
				config *corev1.ConfigMap,
				opts ...client.GetOption,
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
			instanaAgent: instanav1.InstanaAgentRemote{},
		},
		{
			name:     "Should initialize ConfigMap if NotFound error was given and UpdateLifecycleInfo",
			expected: "",
			clientGetterFunc: func(
				_ context.Context,
				_ types.NamespacedName,
				config *corev1.ConfigMap,
				opts ...client.GetOption,
			) error {
				return k8serrors.NewNotFound(schema.GroupResource{}, "asdasd")
			},
			instanaAgent: instanav1.InstanaAgentRemote{},
		},
		{
			name:     "Should initialize empty ConfigMap on error and UpdateLifecycleInfo",
			expected: "",
			clientGetterFunc: func(
				_ context.Context,
				_ types.NamespacedName,
				config *corev1.ConfigMap,
				opts ...client.GetOption,
			) error {
				return errors.New("An expected error occurred")
			},
			instanaAgent: instanav1.InstanaAgentRemote{},
		},
		{
			name:     "Should run with existing ConfigMap.Data field populated and UpdateLifecycleInfo",
			expected: "",
			clientGetterFunc: func(
				_ context.Context,
				_ types.NamespacedName,
				config *corev1.ConfigMap,
				opts ...client.GetOption,
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
			instanaAgent: instanav1.InstanaAgentRemote{ObjectMeta: metav1.ObjectMeta{Generation: 1234}},
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				instanaAgentClient := new(testmocks.MockInstanaAgentClient)

				if test.clientGetterFunc != nil &&
					!strings.Contains(test.name, "NotFound") &&
					!strings.Contains(test.name, "error") {
					instanaAgentClient.On("Get",
						mock.Anything,                        // ctx
						mock.Anything,                        // namespacedName
						mock.AnythingOfType("*v1.ConfigMap"), // config
						mock.Anything,                        // opts
					).Run(func(args mock.Arguments) {
						config := args.Get(2).(*corev1.ConfigMap)
						err := test.clientGetterFunc(nil, types.NamespacedName{}, config)
						if err != nil {
							panic(err)
						}
					}).Return(nil)
				} else {
					instanaAgentClient.On("Get",
						mock.Anything,                        // ctx
						mock.Anything,                        // namespacedName
						mock.AnythingOfType("*v1.ConfigMap"), // config
						mock.Anything,                        // opts
					).Return(test.clientGetterFunc(nil, types.NamespacedName{}, nil))
				}

				instanaAgentClient.On("Apply",
					mock.Anything, // ctx
					mock.Anything, // obj
					mock.Anything, // opts
				).Return(result.Of[client.Object](&unstructured.Unstructured{}, nil))

				dependentLifecycleManager := NewRemoteDependentLifecycleManager(
					ctx,
					&test.instanaAgent,
					instanaAgentClient,
				)
				err := dependentLifecycleManager.UpdateDependentLifecycleInfo([]client.Object{})
				assert.Nil(t, err)
				instanaAgentClient.AssertExpectations(t)
			},
		)
	}
}

// Made with Bob
