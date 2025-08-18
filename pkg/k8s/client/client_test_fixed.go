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

package client

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/instana/instana-agent-operator/internal/testmocks"
	"github.com/instana/instana-agent-operator/pkg/result"
)

func TestInstanaAgentClientApply_Fixed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cm := corev1.ConfigMap{}
	opts := []k8sclient.PatchOption{k8sclient.DryRunAll}
	expectedErr := errors.New("awojsgeoisegoijsdg")

	mockK8sClient := new(testmocks.MockClient)

	// Setup the expected call with proper arguments
	mockK8sClient.On("Patch",
		mock.Anything,                        // ctx
		mock.AnythingOfType("*v1.ConfigMap"), // cm
		k8sclient.Apply,                      // patch type
		mock.Anything,                        // opts - match any options
	).Return(expectedErr)

	client := &instanaAgentClient{
		k8sClient: mockK8sClient,
	}

	actualVal, actualErr := client.Apply(ctx, &cm, opts...).Get()

	assertions := require.New(t)

	assertions.Same(&cm, actualVal)
	assertions.Equal(expectedErr, actualErr)
	mockK8sClient.AssertExpectations(t)
}

func TestInstanaAgentClientGetAsResult_Fixed(t *testing.T) {
	assertions := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	key := types.NamespacedName{
		Namespace: "adsf",
		Name:      "rasdgf",
	}
	obj := &unstructured.Unstructured{}
	opts := &k8sclient.GetOptions{
		Raw: &metav1.GetOptions{
			TypeMeta: metav1.TypeMeta{
				Kind:       "reoisoijd",
				APIVersion: "erifoijsd",
			},
			ResourceVersion: "adsfadsf",
		},
	}

	mockK8sClient := new(testmocks.MockClient)

	// Setup the expected call with proper arguments
	mockK8sClient.On("Get",
		mock.Anything, // ctx
		key,           // key - exact match since it's a simple struct
		mock.AnythingOfType("*unstructured.Unstructured"), // obj - use correct type
		mock.Anything, // opts
	).Return(errors.New("foo"))

	client := &instanaAgentClient{
		k8sClient: mockK8sClient,
	}

	actual := client.GetAsResult(ctx, key, obj, opts, opts)
	assertions.Equal(result.Of[k8sclient.Object](obj, errors.New("foo")), actual)
	mockK8sClient.AssertExpectations(t)
}

func TestInstanaAgentClientExists_Fixed(t *testing.T) {
	for _, test := range []struct {
		name     string
		errOfGet error
		expected result.Result[bool]
	}{
		{
			name:     "crd_exists",
			errOfGet: nil,
			expected: result.OfSuccess(true),
		},
		{
			name: "crd_does_not_exist",
			errOfGet: k8sErrors.NewNotFound(
				schema.GroupResource{
					Group:    "apiextensions.k8s.io",
					Resource: "customresourcedefinitions",
				}, "some-resource",
			),
			expected: result.OfSuccess(false),
		},
		{
			name:     "error_getting_crd",
			errOfGet: errors.New("qwerty"),
			expected: result.OfFailure[bool](errors.New("qwerty")),
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				key := types.NamespacedName{
					Name: "some-resource",
				}
				gvk := schema.GroupVersionKind{
					Group:   "somegroup",
					Version: "v1beta1",
					Kind:    "SomeKind",
				}

				obj := &unstructured.Unstructured{}
				obj.SetGroupVersionKind(gvk)

				mockK8sClient := new(testmocks.MockClient)

				// Setup the expected call with proper arguments
				mockK8sClient.On("Get",
					mock.Anything, // ctx
					key,           // key - exact match since it's a simple struct
					mock.AnythingOfType("*unstructured.Unstructured"), // obj - use correct type
					mock.Anything, // opts - include this to match the actual call
				).Return(test.errOfGet)

				client := NewInstanaAgentClient(mockK8sClient)

				actual := client.Exists(ctx, gvk, key)
				assertions.Equal(test.expected, actual)
				mockK8sClient.AssertExpectations(t)
			},
		)
	}
}

// Made with Bob
