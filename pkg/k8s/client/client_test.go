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

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/instana/instana-agent-operator/mocks"
	"github.com/instana/instana-agent-operator/pkg/result"
)

func TestInstanaAgentClientApply(t *testing.T) {
	ctrl := gomock.NewController(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cm := corev1.ConfigMap{}
	opts := []k8sclient.PatchOption{k8sclient.DryRunAll}
	expectedErr := errors.New("awojsgeoisegoijsdg")

	mockK8sClient := mocks.NewMockClient(ctrl)
	mockK8sClient.EXPECT().Patch(
		gomock.Eq(ctx),
		gomock.Eq(&cm),
		gomock.Eq(k8sclient.Apply),
		gomock.Eq(append(opts, k8sclient.ForceOwnership, k8sclient.FieldOwner("instana-agent-operator"))),
	).Times(1).Return(expectedErr)

	client := instanaAgentClient{
		k8sClient: mockK8sClient,
	}

	actualVal, actualErr := client.Apply(ctx, &cm, opts...).Get()

	assertions := require.New(t)

	assertions.Same(&cm, actualVal)
	assertions.Equal(expectedErr, actualErr)
}

func TestInstanaAgentClientGetAsResult(t *testing.T) {
	assertions := require.New(t)
	ctrl := gomock.NewController(t)
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

	mockK8sClient := mocks.NewMockClient(ctrl)
	mockK8sClient.EXPECT().Get(
		gomock.Eq(ctx),
		gomock.Eq(key),
		gomock.Eq(obj),
		gomock.Eq(opts),
		gomock.Eq(opts),
	).Return(errors.New("foo"))

	instanaAgentClient := instanaAgentClient{
		k8sClient: mockK8sClient,
	}

	actual := instanaAgentClient.GetAsResult(ctx, key, obj, opts, opts)
	assertions.Equal(result.Of[k8sclient.Object](obj, errors.New("foo")), actual)
}

func TestInstanaAgentClientExists(t *testing.T) {
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
				ctrl := gomock.NewController(t)

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

				k8sClient := mocks.NewMockClient(ctrl)
				k8sClient.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(key), gomock.Eq(obj)).Return(test.errOfGet)

				instanaAgentClient := NewInstanaAgentClient(k8sClient)

				actual := instanaAgentClient.Exists(ctx, gvk, key)
				assertions.Equal(test.expected, actual)
			},
		)
	}
}
