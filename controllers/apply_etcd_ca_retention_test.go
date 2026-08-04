/*
(c) Copyright IBM Corp. 2026

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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/internal/mocks"
	backends "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/backends"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	k8ssensordeployment "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/k8s-sensor/deployment"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// agentForETCDCATests returns a minimal agent for the ETCD CA retention cases.
func agentForETCDCATests() *instanav1.InstanaAgent {
	return &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
		},
	}
}

// deploymentWithDiscoveredETCDCA renders the k8sensor Deployment the way the builder
// actually renders it from a discovered CA, rather than hand-building the shape the
// retain path expects. Round tripping through the real builder is the point: if the
// builder ever changes the volume name or the secret it points at, retention stops
// recognising its own output and these tests fail instead of going quietly dead.
func deploymentWithDiscoveredETCDCA(t *testing.T) *appsv1.Deployment {
	t.Helper()

	agent := agentForETCDCATests()
	agent.Spec.Agent.Key = "test-key"
	agent.Spec.Zone.Name = "test-zone"
	agent.Default()

	backendObj := backends.NewK8SensorBackend("", "test-key", "", "test-host", "443")
	built := k8ssensordeployment.NewDeploymentBuilder(
		agent,
		false,
		&status.MockAgentStatusManager{},
		*backendObj,
		nil,
		&k8ssensordeployment.DeploymentContext{ETCDCASecretName: constants.ETCDCASecretName},
	).Build()

	require.True(t, built.IsPresent(), "the builder should produce a Deployment")
	deployment, ok := built.Get().(*appsv1.Deployment)
	require.True(t, ok, "the builder should produce a *appsv1.Deployment")

	return deployment
}

func clientReturningDeployment(deployment *appsv1.Deployment) *mocks.MockInstanaAgentClient {
	mockClient := &mocks.MockInstanaAgentClient{}
	mockClient.On("Get", mock.Anything, mock.Anything, mock.AnythingOfType("*v1.Deployment"), mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			target := args.Get(2).(*appsv1.Deployment)
			*target = *deployment
		})
	// The CA secret is still there unless a test says otherwise. Maybe, because the
	// cases that bail out earlier never get as far as checking it.
	mockClient.On("Get", mock.Anything, mock.Anything, mock.AnythingOfType("*v1.Secret"), mock.Anything).
		Return(nil).
		Maybe()
	return mockClient
}

// TestETCDCAReconcileFailsWhenDeploymentUnreadable covers the case where the applied
// state itself cannot be read. Treating that as "nothing to retain" would strip the CA
// and roll the pod, which is the same unknown-means-absent conflation this path exists
// to avoid, so the error is reported and the reconcile retried instead.
func TestETCDCAReconcileFailsWhenDeploymentUnreadable(t *testing.T) {
	agent := agentForETCDCATests()
	ctx := context.Background()
	logger := zap.New()

	discoverETCD := func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
		return nil, assert.AnError
	}

	mockClient := &mocks.MockInstanaAgentClient{}
	mockClient.On("Get", mock.Anything, mock.Anything, mock.AnythingOfType("*v1.Deployment"), mock.Anything).
		Return(fmt.Errorf("cache not synced"))

	deploymentContext, err := CreateDeploymentContext(
		ctx,
		mockClient,
		agent,
		false,
		logger,
		discoverETCD,
	)

	require.Error(t, err, "an unreadable Deployment should be reported, not guessed away")
	assert.Nil(t, deploymentContext)
	mockClient.AssertExpectations(t)
}

// TestETCDCANotRetainedWhenSecretIsGone covers a secret deleted while discovery is
// degraded. Retaining it would leave the Deployment referencing a secret that no longer
// exists, which the k8sensor cannot use and which used to be pinned indefinitely,
// because the inconclusive path never reaches the discoverer's own CA check.
func TestETCDCANotRetainedWhenSecretIsGone(t *testing.T) {
	agent := agentForETCDCATests()
	ctx := context.Background()
	logger := zap.New()

	discoverETCD := func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
		return &DiscoveredETCDTargets{Indeterminate: true}, nil
	}

	mockClient := &mocks.MockInstanaAgentClient{}
	mockClient.On("Get", mock.Anything, mock.Anything, mock.AnythingOfType("*v1.Deployment"), mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			*(args.Get(2).(*appsv1.Deployment)) = *deploymentWithDiscoveredETCDCA(t)
		})
	mockClient.On("Get", mock.Anything, mock.Anything, mock.AnythingOfType("*v1.Secret"), mock.Anything).
		Return(apierrors.NewNotFound(schema.GroupResource{}, constants.ETCDCASecretName))

	deploymentContext, err := CreateDeploymentContext(
		ctx,
		mockClient,
		agent,
		false,
		logger,
		discoverETCD,
	)

	require.NoError(t, err)
	assert.Nil(t, deploymentContext, "a deleted CA secret must not stay referenced")
	mockClient.AssertExpectations(t)
}

// TestETCDCARetainedWhenSecretLookupFails covers the ambiguous case: if the secret
// cannot be read, that is unknown rather than absent, so the applied CA is kept.
func TestETCDCARetainedWhenSecretLookupFails(t *testing.T) {
	agent := agentForETCDCATests()
	ctx := context.Background()
	logger := zap.New()

	discoverETCD := func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
		return &DiscoveredETCDTargets{Indeterminate: true}, nil
	}

	mockClient := &mocks.MockInstanaAgentClient{}
	mockClient.On("Get", mock.Anything, mock.Anything, mock.AnythingOfType("*v1.Deployment"), mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			*(args.Get(2).(*appsv1.Deployment)) = *deploymentWithDiscoveredETCDCA(t)
		})
	mockClient.On("Get", mock.Anything, mock.Anything, mock.AnythingOfType("*v1.Secret"), mock.Anything).
		Return(assert.AnError)

	deploymentContext, err := CreateDeploymentContext(
		ctx,
		mockClient,
		agent,
		false,
		logger,
		discoverETCD,
	)

	require.NoError(t, err)
	require.NotNil(t, deploymentContext, "an unreadable secret is unknown, not absent")
	assert.Equal(t, constants.ETCDCASecretName, deploymentContext.ETCDCASecretName)
	mockClient.AssertExpectations(t)
}

// TestETCDCARetainedWhenDiscoveryFails covers a transient discovery failure. An error
// means the state is unknown, not that there is nothing to monitor, so the CA that is
// already applied has to survive rather than be stripped and the pod rolled.
func TestETCDCARetainedWhenDiscoveryFails(t *testing.T) {
	agent := agentForETCDCATests()
	ctx := context.Background()
	logger := zap.New()

	discoverETCD := func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
		return nil, assert.AnError
	}

	mockClient := clientReturningDeployment(deploymentWithDiscoveredETCDCA(t))

	deploymentContext, err := CreateDeploymentContext(
		ctx,
		mockClient,
		agent,
		false,
		logger,
		discoverETCD,
	)

	require.NoError(t, err)
	require.NotNil(t, deploymentContext, "a failed discovery must not drop the applied CA")
	assert.Equal(t, constants.ETCDCASecretName, deploymentContext.ETCDCASecretName)
	mockClient.AssertExpectations(t)
}

// TestETCDCARetainedWhenDiscoveryInconclusive covers the other transient case, where
// the etcd service is still there but no endpoint is ready on this pass.
func TestETCDCARetainedWhenDiscoveryInconclusive(t *testing.T) {
	agent := agentForETCDCATests()
	ctx := context.Background()
	logger := zap.New()

	discoverETCD := func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
		return &DiscoveredETCDTargets{Indeterminate: true}, nil
	}

	mockClient := clientReturningDeployment(deploymentWithDiscoveredETCDCA(t))

	deploymentContext, err := CreateDeploymentContext(
		ctx,
		mockClient,
		agent,
		false,
		logger,
		discoverETCD,
	)

	require.NoError(t, err)
	require.NotNil(t, deploymentContext, "an inconclusive discovery must not drop the applied CA")
	assert.Equal(t, constants.ETCDCASecretName, deploymentContext.ETCDCASecretName)
	mockClient.AssertExpectations(t)
}

// TestETCDCAClearedWhenPositivelyGone makes sure retention does not pin the CA
// forever. When discovery successfully reports that the CA secret is no longer there,
// it is dropped so the k8sensor stops pointing at a file that is not mounted.
func TestETCDCAClearedWhenPositivelyGone(t *testing.T) {
	agent := agentForETCDCATests()
	ctx := context.Background()
	logger := zap.New()

	discoverETCD := func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
		return &DiscoveredETCDTargets{
			Targets: []string{"https://etcd-1:2379/metrics"},
			CAFound: false,
		}, nil
	}

	// No Get is expected at all: a positive result never consults the live Deployment
	mockClient := &mocks.MockInstanaAgentClient{}

	deploymentContext, err := CreateDeploymentContext(
		ctx,
		mockClient,
		agent,
		false,
		logger,
		discoverETCD,
	)

	require.NoError(t, err)
	assert.Nil(t, deploymentContext, "a positively absent CA should be dropped")
	mockClient.AssertExpectations(t)
}

// TestETCDCANotRetainedWithoutExistingDeployment covers the first reconcile on a fresh
// install, where discovery fails before anything has been applied to fall back on.
func TestETCDCANotRetainedWithoutExistingDeployment(t *testing.T) {
	agent := agentForETCDCATests()
	ctx := context.Background()
	logger := zap.New()

	discoverETCD := func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
		return nil, assert.AnError
	}

	mockClient := &mocks.MockInstanaAgentClient{}
	mockClient.On("Get", mock.Anything, mock.Anything, mock.AnythingOfType("*v1.Deployment"), mock.Anything).
		Return(apierrors.NewNotFound(schema.GroupResource{}, ""))

	deploymentContext, err := CreateDeploymentContext(
		ctx,
		mockClient,
		agent,
		false,
		logger,
		discoverETCD,
	)

	require.NoError(t, err)
	assert.Nil(t, deploymentContext, "there is nothing to retain on a fresh install")
	mockClient.AssertExpectations(t)
}

// TestETCDCANotRetainedForCollidingCRMountPath is the regression test for a CR that
// points its own CA mount path at the same place the operator mounts a discovered one.
// The rendered ETCD_CA_FILE is then byte identical to the discovered case while the pod
// carries no etcd-ca volume at all, so matching on the env var used to retain a CA that
// was never there, add a secret volume, and roll the pod on every transient failure.
func TestETCDCANotRetainedForCollidingCRMountPath(t *testing.T) {
	agent := agentForETCDCATests()
	agent.Spec.K8sSensor.ETCD.CA.MountPath = constants.ETCDCAMountPath
	ctx := context.Background()
	logger := zap.New()

	// Rendered with no discovered CA: the env var matches, but there is no volume
	applied := deploymentWithDiscoveredETCDCA(t)
	applied.Spec.Template.Spec.Volumes = nil

	discoverETCD := func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
		return nil, assert.AnError
	}

	mockClient := clientReturningDeployment(applied)

	deploymentContext, err := CreateDeploymentContext(
		ctx,
		mockClient,
		agent,
		false,
		logger,
		discoverETCD,
	)

	require.NoError(t, err)
	assert.Nil(
		t,
		deploymentContext,
		"a CR mount path that collides with the discovered one must not look like a discovered CA",
	)
	mockClient.AssertExpectations(t)
}

// TestETCDCANotRetainedForCRConfiguredCA makes sure retention only ever brings back
// the discovered CA. A CA configured on the CR is rendered from the spec on every
// reconcile, so it must not be mistaken for a discovered one.
func TestETCDCANotRetainedForCRConfiguredCA(t *testing.T) {
	agent := agentForETCDCATests()
	agent.Spec.K8sSensor.ETCD.CA.SecretName = "my-own-etcd-ca"
	ctx := context.Background()
	logger := zap.New()

	applied := deploymentWithDiscoveredETCDCA(t)

	discoverETCD := func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
		return nil, assert.AnError
	}

	mockClient := clientReturningDeployment(applied)

	deploymentContext, err := CreateDeploymentContext(
		ctx,
		mockClient,
		agent,
		false,
		logger,
		discoverETCD,
	)

	require.NoError(t, err)
	assert.Nil(
		t,
		deploymentContext,
		"a CR configured CA should not be retained as a discovered one",
	)
	mockClient.AssertExpectations(t)
}
