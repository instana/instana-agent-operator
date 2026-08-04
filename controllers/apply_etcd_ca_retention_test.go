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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/internal/mocks"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
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

// deploymentWithDiscoveredETCDCA returns a k8sensor Deployment carrying the ETCD CA
// settings a previous reconcile would have applied from discovery.
func deploymentWithDiscoveredETCDCA() *appsv1.Deployment {
	return &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: constants.ETCDCAVolumeName,
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{ // pragma: allowlist secret
									SecretName: constants.ETCDCASecretName,
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name: constants.ContainerK8Sensor,
							Env: []corev1.EnvVar{
								{
									Name:  constants.EnvETCDCAFile,
									Value: constants.ETCDCAMountPath + "/ca.crt",
								},
							},
						},
					},
				},
			},
		},
	}
}

func clientReturningDeployment(deployment *appsv1.Deployment) *mocks.MockInstanaAgentClient {
	mockClient := &mocks.MockInstanaAgentClient{}
	mockClient.On("Get", mock.Anything, mock.Anything, mock.AnythingOfType("*v1.Deployment"), mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			target := args.Get(2).(*appsv1.Deployment)
			*target = *deployment
		})
	return mockClient
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

	mockClient := clientReturningDeployment(deploymentWithDiscoveredETCDCA())

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

	mockClient := clientReturningDeployment(deploymentWithDiscoveredETCDCA())

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
	applied := deploymentWithDiscoveredETCDCA()
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

	applied := deploymentWithDiscoveredETCDCA()

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
