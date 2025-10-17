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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/internal/mocks"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/result"
)

func TestCreateDeploymentContext_SimplifiedTests(t *testing.T) {
	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
		},
	}

	ctx := context.Background()
	logger := zap.New()

	t.Run("OpenShift creates service CA", func(t *testing.T) {
		mockClient := &mocks.MockInstanaAgentClient{}

		// Mock the Apply method for CreateServiceCAConfigMap
		mockClient.On("Apply", mock.Anything, mock.AnythingOfType("*v1.ConfigMap"), mock.Anything).Return(result.OfSuccess[client.Object](nil))

		// Mock ETCD discover function (won't be called for OpenShift)
		mockDiscoverETCD := func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
			return nil, nil
		}

		deploymentContext, err := CreateDeploymentContext(ctx, mockClient, agent, true, logger, mockDiscoverETCD)

		require.NoError(t, err)
		require.NotNil(t, deploymentContext)
		assert.Empty(t, deploymentContext.ETCDCASecretName)
		mockClient.AssertExpectations(t)
	})

	t.Run("Vanilla K8s with no ETCD returns nil", func(t *testing.T) {
		mockClient := &mocks.MockInstanaAgentClient{}

		// Mock ETCD discover function that returns nil
		mockDiscoverETCD := func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
			return nil, nil
		}

		deploymentContext, err := CreateDeploymentContext(ctx, mockClient, agent, false, logger, mockDiscoverETCD)

		require.NoError(t, err)
		assert.Nil(t, deploymentContext)
		mockClient.AssertExpectations(t)
	})

	t.Run("Vanilla K8s with ETCD creates deployment context", func(t *testing.T) {
		mockClient := &mocks.MockInstanaAgentClient{}

		// Mock ETCD discover function that returns targets
		mockDiscoverETCD := func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
			return &DiscoveredETCDTargets{
				Targets: []string{"https://etcd-1:2379/metrics", "https://etcd-2:2379/metrics"},
				CAFound: true,
			}, nil
		}

		mockClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(apierrors.NewNotFound(schema.GroupResource{}, ""))

		deploymentContext, err := CreateDeploymentContext(ctx, mockClient, agent, false, logger, mockDiscoverETCD)

		require.NoError(t, err)
		require.NotNil(t, deploymentContext)
		assert.Equal(t, []string{"https://etcd-1:2379/metrics", "https://etcd-2:2379/metrics"}, deploymentContext.DiscoveredETCDTargets)
		assert.Equal(t, constants.ETCDCASecretName, deploymentContext.ETCDCASecretName)
		mockClient.AssertExpectations(t)
	})

	t.Run("Deployment exists with same ETCD targets - no update", func(t *testing.T) {
		mockClient := &mocks.MockInstanaAgentClient{}

		// Mock ETCD discover function
		mockDiscoverETCD := func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
			return &DiscoveredETCDTargets{
				Targets: []string{"https://etcd-1:2379/metrics", "https://etcd-2:2379/metrics"},
				CAFound: true,
			}, nil
		}

		// Create existing deployment with same targets
		existingDeployment := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: constants.ContainerK8Sensor,
								Env: []corev1.EnvVar{
									{
										Name:  constants.EnvETCDTargets,
										Value: "https://etcd-1:2379/metrics,https://etcd-2:2379/metrics",
									},
								},
							},
						},
					},
				},
			},
		}

		mockClient.On("Get", mock.Anything, mock.Anything, mock.AnythingOfType("*v1.Deployment"), mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			deployment := args.Get(2).(*appsv1.Deployment)
			*deployment = *existingDeployment
		})

		deploymentContext, err := CreateDeploymentContext(ctx, mockClient, agent, false, logger, mockDiscoverETCD)

		require.NoError(t, err)
		assert.Nil(t, deploymentContext) // Should return nil when no update needed
		mockClient.AssertExpectations(t)
	})

	t.Run("Deployment exists with different ETCD targets - update needed", func(t *testing.T) {
		mockClient := &mocks.MockInstanaAgentClient{}

		// Mock ETCD discover function with new targets
		mockDiscoverETCD := func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
			return &DiscoveredETCDTargets{
				Targets: []string{"https://etcd-3:2379/metrics", "https://etcd-4:2379/metrics"},
				CAFound: true,
			}, nil
		}

		// Create existing deployment with different targets
		existingDeployment := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: constants.ContainerK8Sensor,
								Env: []corev1.EnvVar{
									{
										Name:  constants.EnvETCDTargets,
										Value: "https://etcd-1:2379/metrics,https://etcd-2:2379/metrics",
									},
								},
							},
						},
					},
				},
			},
		}

		mockClient.On("Get", mock.Anything, mock.Anything, mock.AnythingOfType("*v1.Deployment"), mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			deployment := args.Get(2).(*appsv1.Deployment)
			*deployment = *existingDeployment
		})

		deploymentContext, err := CreateDeploymentContext(ctx, mockClient, agent, false, logger, mockDiscoverETCD)

		require.NoError(t, err)
		require.NotNil(t, deploymentContext)
		assert.Equal(t, []string{"https://etcd-3:2379/metrics", "https://etcd-4:2379/metrics"}, deploymentContext.DiscoveredETCDTargets)
		assert.Equal(t, constants.ETCDCASecretName, deploymentContext.ETCDCASecretName)
		mockClient.AssertExpectations(t)
	})

	t.Run("Error handling in service CA creation", func(t *testing.T) {
		mockClient := &mocks.MockInstanaAgentClient{}

		// Mock Apply to return an error
		mockClient.On("Apply", mock.Anything, mock.AnythingOfType("*v1.ConfigMap"), mock.Anything).Return(result.OfFailure[client.Object](assert.AnError))

		// Mock ETCD discover function (won't be called for OpenShift)
		mockDiscoverETCD := func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
			return nil, nil
		}

		deploymentContext, err := CreateDeploymentContext(ctx, mockClient, agent, true, logger, mockDiscoverETCD)

		require.NoError(t, err) // Function continues on error
		assert.Nil(t, deploymentContext)
		mockClient.AssertExpectations(t)
	})

	t.Run("Error handling in ETCD discovery", func(t *testing.T) {
		mockClient := &mocks.MockInstanaAgentClient{}

		// Mock ETCD discover function that returns error
		mockDiscoverETCD := func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
			return nil, assert.AnError
		}

		deploymentContext, err := CreateDeploymentContext(ctx, mockClient, agent, false, logger, mockDiscoverETCD)

		require.NoError(t, err) // Function continues on error
		assert.Nil(t, deploymentContext)
		mockClient.AssertExpectations(t)
	})
}
