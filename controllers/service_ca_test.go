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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/internal/mocks"
	"github.com/instana/instana-agent-operator/pkg/result"
)

func TestCreateServiceCAConfigMap(t *testing.T) {
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

	ctx := context.Background()

	t.Run("Should create service CA ConfigMap successfully", func(t *testing.T) {
		mockClient := &mocks.MockInstanaAgentClient{}

		// Mock successful Apply
		mockClient.On("Apply", mock.Anything, mock.AnythingOfType("*v1.ConfigMap"), mock.Anything).Return(result.OfSuccess[client.Object](nil))

		// Create a mock reconciler to get the logger
		reconciler := &InstanaAgentReconciler{
			client: mockClient,
		}

		log := reconciler.loggerFor(ctx, agent)
		err := CreateServiceCAConfigMap(ctx, mockClient, agent, log)

		require.NoError(t, err, "Should not return an error")
		mockClient.AssertExpectations(t)
	})

	t.Run("Should handle Apply error gracefully", func(t *testing.T) {
		mockClient := &mocks.MockInstanaAgentClient{}

		// Mock Apply failure
		mockClient.On("Apply", mock.Anything, mock.AnythingOfType("*v1.ConfigMap"), mock.Anything).Return(result.OfFailure[client.Object](assert.AnError))

		// Create a mock reconciler to get the logger
		reconciler := &InstanaAgentReconciler{
			client: mockClient,
		}

		log := reconciler.loggerFor(ctx, agent)
		err := CreateServiceCAConfigMap(ctx, mockClient, agent, log)

		require.Error(t, err, "Should return an error")
		mockClient.AssertExpectations(t)
	})
}

func TestCreateServiceCAConfigMapUpdate(t *testing.T) {
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

	ctx := context.Background()

	t.Run("Should update existing ConfigMap successfully", func(t *testing.T) {
		mockClient := &mocks.MockInstanaAgentClient{}

		// Mock successful Apply (simulating update scenario)
		mockClient.On("Apply", mock.Anything, mock.AnythingOfType("*v1.ConfigMap"), mock.Anything).Return(result.OfSuccess[client.Object](nil))

		// Create a mock reconciler to get the logger
		reconciler := &InstanaAgentReconciler{
			client: mockClient,
		}

		log := reconciler.loggerFor(ctx, agent)
		err := CreateServiceCAConfigMap(ctx, mockClient, agent, log)

		require.NoError(t, err, "Should not return an error")
		mockClient.AssertExpectations(t)
	})
}
