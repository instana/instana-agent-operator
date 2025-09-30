package controllers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

		deploymentContext, err := CreateDeploymentContext(ctx, mockClient, agent, true, logger, nil)

		require.NoError(t, err)
		require.NotNil(t, deploymentContext)
		assert.Equal(t, constants.ServiceCAConfigMapName, deploymentContext.ETCDCASecretName)
		mockClient.AssertExpectations(t)
	})

	t.Run("Vanilla K8s with no ETCD returns nil", func(t *testing.T) {
		mockClient := &mocks.MockInstanaAgentClient{}

		deploymentContext, err := CreateDeploymentContext(ctx, mockClient, agent, false, logger, nil)

		require.NoError(t, err)
		assert.Nil(t, deploymentContext)
		mockClient.AssertExpectations(t)
	})

	t.Run("Vanilla K8s with ETCD creates deployment context", func(t *testing.T) {
		mockClient := &mocks.MockInstanaAgentClient{}

		discoveredETCD := &DiscoveredETCDTargets{
			Targets: []string{"https://etcd-1:2379/metrics", "https://etcd-2:2379/metrics"},
			CAFound: true,
		}
		mockClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(apierrors.NewNotFound(schema.GroupResource{}, ""))

		deploymentContext, err := CreateDeploymentContext(ctx, mockClient, agent, false, logger, discoveredETCD)

		require.NoError(t, err)
		require.NotNil(t, deploymentContext)
		assert.Equal(t, []string{"https://etcd-1:2379/metrics", "https://etcd-2:2379/metrics"}, deploymentContext.DiscoveredETCDTargets)
		assert.Equal(t, constants.ETCDCASecretName, deploymentContext.ETCDCASecretName)
		mockClient.AssertExpectations(t)
	})

	t.Run("Error handling in service CA creation", func(t *testing.T) {
		mockClient := &mocks.MockInstanaAgentClient{}

		// Mock Apply to return an error
		mockClient.On("Apply", mock.Anything, mock.AnythingOfType("*v1.ConfigMap"), mock.Anything).Return(result.OfFailure[client.Object](assert.AnError))

		deploymentContext, err := CreateDeploymentContext(ctx, mockClient, agent, true, logger, nil)

		require.NoError(t, err) // Function continues on error
		assert.Nil(t, deploymentContext)
		mockClient.AssertExpectations(t)
	})
}
