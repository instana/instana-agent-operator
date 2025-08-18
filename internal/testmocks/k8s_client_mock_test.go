/*
 * (c) Copyright IBM Corp. 2025
 */

package testmocks

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestMockClient(t *testing.T) {
	// Create a new mock client
	mockClient := new(MockClient)

	// Set up the context and test objects
	ctx := context.Background()
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-configmap",
			Namespace: "default",
		},
		Data: map[string]string{
			"key": "value",
		},
	}
	namespacedName := types.NamespacedName{
		Name:      "test-configmap",
		Namespace: "default",
	}

	// Set up expectations
	expectedError := errors.New("test error")
	mockClient.On(
		"Get",
		mock.Anything,
		namespacedName,
		mock.AnythingOfType("*v1.ConfigMap"),
	).Return(expectedError)

	// Call the method we're testing
	err := mockClient.Get(ctx, namespacedName, configMap)

	// Assertions
	assert.Equal(t, expectedError, err)
	mockClient.AssertExpectations(t)
}

func TestMockClientPatch(t *testing.T) {
	// Create a new mock client
	mockClient := new(MockClient)

	// Set up the context and test objects
	ctx := context.Background()
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-configmap",
			Namespace: "default",
		},
		Data: map[string]string{
			"key": "value",
		},
	}

	// Set up expectations
	expectedError := errors.New("patch error")
	mockClient.On(
		"Patch",
		mock.Anything,
		mock.AnythingOfType("*v1.ConfigMap"),
		client.Apply,
		mock.Anything,
	).Return(expectedError)

	// Call the method we're testing
	err := mockClient.Patch(ctx, configMap, client.Apply, client.FieldOwner("test"))

	// Assertions
	assert.Equal(t, expectedError, err)
	mockClient.AssertExpectations(t)
}

// Made with Bob
