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

package controllers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/namespaces"
	"github.com/instana/instana-agent-operator/pkg/result"
)

// mockInstanaClient is a simple mock implementation for testing
type mockInstanaClient struct {
	fakeClient client.Client
}

// Apply implements a basic Apply method that returns a successful result
func (m *mockInstanaClient) Apply(
	ctx context.Context,
	obj client.Object,
	opts ...client.PatchOption,
) result.Result[client.Object] {
	// For testing, we'll simulate a successful apply by creating/updating the object
	err := m.fakeClient.Create(ctx, obj)
	if err != nil {
		// If create fails, try update
		err = m.fakeClient.Update(ctx, obj)
	}

	if err != nil {
		return result.Of(obj, err)
	}
	return result.Of(obj, nil)
}

// Implement other required methods as no-ops for testing
func (m *mockInstanaClient) Exists(
	ctx context.Context,
	gvk schema.GroupVersionKind,
	key client.ObjectKey,
) result.Result[bool] {
	return result.Of(false, nil)
}

func (m *mockInstanaClient) DeleteAllInTimeLimit(
	ctx context.Context,
	objects []client.Object,
	timeout time.Duration,
	waitTime time.Duration,
	opts ...client.DeleteOption,
) result.Result[[]client.Object] {
	return result.Of(objects, nil)
}

func (m *mockInstanaClient) Get(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
	return m.fakeClient.Get(ctx, key, obj, opts...)
}

func (m *mockInstanaClient) GetAsResult(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) result.Result[client.Object] {
	err := m.fakeClient.Get(ctx, key, obj, opts...)
	return result.Of(obj, err)
}

func (m *mockInstanaClient) Status() client.SubResourceWriter {
	return m.fakeClient.Status()
}

func (m *mockInstanaClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return m.fakeClient.Patch(ctx, obj, patch, opts...)
}

func (m *mockInstanaClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	return m.fakeClient.Delete(ctx, obj, opts...)
}

func (m *mockInstanaClient) GetNamespacesWithLabels(ctx context.Context) (namespaces.NamespacesDetails, error) {
	return namespaces.NamespacesDetails{}, nil
}

func TestCreateServiceCAConfigMap(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = instanav1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	// Test cases
	testCases := []struct {
		name         string
		agent        *instanav1.InstanaAgent
		expectError  bool
		validateFunc func(t *testing.T, client client.Client, agent *instanav1.InstanaAgent)
	}{
		{
			name: "Should create service CA ConfigMap successfully",
			agent: &instanav1.InstanaAgent{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "instana.io/v1",
					Kind:       "InstanaAgent",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "test-namespace",
					UID:       "test-uid",
				},
			},
			expectError: false,
			validateFunc: func(t *testing.T, client client.Client, agent *instanav1.InstanaAgent) {
				// Verify ConfigMap was created
				configMap := &corev1.ConfigMap{}
				err := client.Get(context.Background(), types.NamespacedName{
					Namespace: agent.Namespace,
					Name:      constants.ServiceCAConfigMapName,
				}, configMap)

				require.NoError(t, err, "ConfigMap should be created")
				assert.Equal(t, constants.ServiceCAConfigMapName, configMap.Name, "ConfigMap name should match")
				assert.Equal(t, agent.Namespace, configMap.Namespace, "ConfigMap namespace should match")

				// Verify annotations
				assert.Contains(t, configMap.Annotations, constants.OpenShiftInjectCABundleAnnotation, "Should have inject-cabundle annotation")
				assert.Equal(
					t,
					"true",
					configMap.Annotations[constants.OpenShiftInjectCABundleAnnotation],
					"Annotation value should be 'true'",
				)

				// Verify owner references
				require.Len(t, configMap.OwnerReferences, 1, "Should have one owner reference")
				ownerRef := configMap.OwnerReferences[0]
				assert.Equal(t, agent.APIVersion, ownerRef.APIVersion, "Owner reference API version should match")
				assert.Equal(t, agent.Kind, ownerRef.Kind, "Owner reference kind should match")
				assert.Equal(t, agent.Name, ownerRef.Name, "Owner reference name should match")
				assert.Equal(t, agent.UID, ownerRef.UID, "Owner reference UID should match")
				assert.True(t, *ownerRef.Controller, "Owner reference controller should be true")

				// Verify data is empty (fake client may set Data to nil for empty maps)
				if configMap.Data != nil {
					assert.Empty(t, configMap.Data, "Data should be empty")
				}
			},
		},
		{
			name: "Should handle agent with minimal metadata",
			agent: &instanav1.InstanaAgent{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "instana.io/v1",
					Kind:       "InstanaAgent",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "minimal-agent",
					Namespace: "minimal-namespace",
					UID:       "minimal-uid",
				},
			},
			expectError: false,
			validateFunc: func(t *testing.T, client client.Client, agent *instanav1.InstanaAgent) {
				// Verify ConfigMap was created with minimal metadata
				configMap := &corev1.ConfigMap{}
				err := client.Get(context.Background(), types.NamespacedName{
					Namespace: agent.Namespace,
					Name:      constants.ServiceCAConfigMapName,
				}, configMap)

				require.NoError(t, err, "ConfigMap should be created")
				assert.Equal(t, agent.Namespace, configMap.Namespace, "ConfigMap namespace should match")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a fake client
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

			// Create a simple mock that implements the basic Apply interface
			mockClient := &mockInstanaClient{
				fakeClient: fakeClient,
			}

			// Create a mock reconciler to test the method
			reconciler := &InstanaAgentReconciler{
				client: mockClient,
			}

			// Test the free function
			ctx := context.Background()
			log := reconciler.loggerFor(ctx, tc.agent)
			err := CreateServiceCAConfigMap(ctx, reconciler.client, tc.agent, log)

			// Verify error expectation
			if tc.expectError {
				assert.Error(t, err, "Should return an error")
			} else {
				assert.NoError(t, err, "Should not return an error")

				// Run validation function if provided
				if tc.validateFunc != nil {
					tc.validateFunc(t, fakeClient, tc.agent)
				}
			}
		})
	}
}

func TestCreateServiceCAConfigMapUpdate(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = instanav1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	agent := &instanav1.InstanaAgent{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "instana.io/v1",
			Kind:       "InstanaAgent",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
			UID:       "test-uid",
		},
	}

	// Create existing ConfigMap with different content
	existingConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.ServiceCAConfigMapName,
			Namespace: agent.Namespace,
			Annotations: map[string]string{
				"some.other.annotation": "value",
			},
		},
		Data: map[string]string{
			"existing-key": "existing-value",
		},
	}

	// Create client with existing ConfigMap
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(existingConfigMap).
		Build()

	// Create mock client and reconciler
	mockClient := &mockInstanaClient{
		fakeClient: fakeClient,
	}
	reconciler := &InstanaAgentReconciler{
		client: mockClient,
	}

	// Test the free function
	ctx := context.Background()
	log := reconciler.loggerFor(ctx, agent)
	err := CreateServiceCAConfigMap(ctx, reconciler.client, agent, log)

	// Verify no error
	require.NoError(t, err, "Should not return an error")

	// Verify ConfigMap was updated
	configMap := &corev1.ConfigMap{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Namespace: agent.Namespace,
		Name:      constants.ServiceCAConfigMapName,
	}, configMap)

	require.NoError(t, err, "ConfigMap should exist")

	// Verify the ConfigMap has the correct annotation (should be updated/merged)
	assert.Contains(t, configMap.Annotations, constants.OpenShiftInjectCABundleAnnotation, "Should have inject-cabundle annotation")
	assert.Equal(
		t,
		"true",
		configMap.Annotations[constants.OpenShiftInjectCABundleAnnotation],
		"Annotation value should be 'true'",
	)

	// Verify owner references were set
	require.Len(t, configMap.OwnerReferences, 1, "Should have one owner reference")
	ownerRef := configMap.OwnerReferences[0]
	assert.Equal(t, agent.Name, ownerRef.Name, "Owner reference name should match")
}
