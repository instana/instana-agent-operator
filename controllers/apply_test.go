package controllers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
)

// MockDeploymentContextDependenciesSimple is a simplified mock for testing the pure function
type MockDeploymentContextDependenciesSimple struct {
	mock.Mock
}

func (m *MockDeploymentContextDependenciesSimple) CreateServiceCAConfigMap(ctx context.Context, agent *instanav1.InstanaAgent, logger logr.Logger) error {
	args := m.Called(ctx, agent, logger)
	return args.Error(0)
}

func (m *MockDeploymentContextDependenciesSimple) DiscoverETCDEndpoints(ctx context.Context, agent *instanav1.InstanaAgent, logger logr.Logger) (*DiscoveredETCDTargets, error) {
	args := m.Called(ctx, agent, logger)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DiscoveredETCDTargets), args.Error(1)
}

func (m *MockDeploymentContextDependenciesSimple) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	args := m.Called(ctx, key, obj)
	return args.Error(0)
}

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
		mockDeps := &MockDeploymentContextDependenciesSimple{}
		mockDeps.On("CreateServiceCAConfigMap", mock.Anything, agent, mock.Anything).Return(nil)

		deploymentContext, err := CreateDeploymentContext(ctx, mockDeps, agent, true, logger)

		require.NoError(t, err)
		require.NotNil(t, deploymentContext)
		assert.Equal(t, constants.ServiceCAConfigMapName, deploymentContext.ETCDCASecretName)
		mockDeps.AssertExpectations(t)
	})

	t.Run("Vanilla K8s with no ETCD returns nil", func(t *testing.T) {
		mockDeps := &MockDeploymentContextDependenciesSimple{}
		mockDeps.On("DiscoverETCDEndpoints", mock.Anything, agent, mock.Anything).Return(nil, nil)

		deploymentContext, err := CreateDeploymentContext(ctx, mockDeps, agent, false, logger)

		require.NoError(t, err)
		assert.Nil(t, deploymentContext)
		mockDeps.AssertExpectations(t)
	})

	t.Run("Vanilla K8s with ETCD creates deployment context", func(t *testing.T) {
		mockDeps := &MockDeploymentContextDependenciesSimple{}
		mockDeps.On("DiscoverETCDEndpoints", mock.Anything, agent, mock.Anything).Return(&DiscoveredETCDTargets{
			Targets: []string{"https://etcd-1:2379/metrics", "https://etcd-2:2379/metrics"},
			CAFound: true,
		}, nil)
		mockDeps.On("Get", mock.Anything, mock.Anything, mock.Anything).Return(apierrors.NewNotFound(schema.GroupResource{}, ""))

		deploymentContext, err := CreateDeploymentContext(ctx, mockDeps, agent, false, logger)

		require.NoError(t, err)
		require.NotNil(t, deploymentContext)
		assert.Equal(t, []string{"https://etcd-1:2379/metrics", "https://etcd-2:2379/metrics"}, deploymentContext.DiscoveredETCDTargets)
		assert.Equal(t, constants.ETCDCASecretName, deploymentContext.ETCDCASecretName)
		mockDeps.AssertExpectations(t)
	})

	t.Run("Error handling in service CA creation", func(t *testing.T) {
		mockDeps := &MockDeploymentContextDependenciesSimple{}
		mockDeps.On("CreateServiceCAConfigMap", mock.Anything, agent, mock.Anything).Return(assert.AnError)

		deploymentContext, err := CreateDeploymentContext(ctx, mockDeps, agent, true, logger)

		require.NoError(t, err) // Function continues on error
		assert.Nil(t, deploymentContext)
		mockDeps.AssertExpectations(t)
	})

	t.Run("Error handling in ETCD discovery", func(t *testing.T) {
		mockDeps := &MockDeploymentContextDependenciesSimple{}
		mockDeps.On("DiscoverETCDEndpoints", mock.Anything, agent, mock.Anything).Return(nil, assert.AnError)

		deploymentContext, err := CreateDeploymentContext(ctx, mockDeps, agent, false, logger)

		require.NoError(t, err) // Function continues on error
		assert.Nil(t, deploymentContext)
		mockDeps.AssertExpectations(t)
	})
}
