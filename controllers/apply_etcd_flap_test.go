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
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/internal/mocks"
	backends "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/backends"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	k8ssensordeployment "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/k8s-sensor/deployment"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/status"
)

// renderK8sSensorDeployment builds the k8s-sensor Deployment the way applyResources
// does, so the test sees exactly what would be sent to the server-side apply.
func renderK8sSensorDeployment(
	t *testing.T,
	agent *instanav1.InstanaAgent,
	deploymentContext *k8ssensordeployment.DeploymentContext,
) *appsv1.Deployment {
	t.Helper()

	backend := backends.NewK8SensorBackend("", "test-key", "", "test-host", "443")
	built := k8ssensordeployment.NewDeploymentBuilder(
		agent,
		false,
		&status.MockAgentStatusManager{},
		*backend,
		nil,
		deploymentContext,
	).Build()

	require.True(t, built.IsPresent(), "deployment builder should produce a Deployment")

	deployment, ok := built.Get().(*appsv1.Deployment)
	require.True(t, ok, "builder should produce a *appsv1.Deployment")
	return deployment
}

// etcdTargetsEnvOf returns the ETCD_TARGETS value on the k8sensor container, or ""
// when the env var is absent.
func etcdTargetsEnvOf(deployment *appsv1.Deployment) string {
	for _, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name != constants.ContainerK8Sensor {
			continue
		}
		for _, env := range container.Env {
			if env.Name == constants.EnvETCDTargets {
				return env.Value
			}
		}
	}
	return ""
}

// TestETCDTargetsRemainStableAcrossReconciles reproduces INSTA-105465: on the
// reconcile after the targets were applied, discovery finds the same targets, and
// the rendered Deployment used to drop ETCD_TARGETS entirely. Server-side apply then
// stripped the env var and rolled the pod, and the next reconcile added it back,
// leaving the k8sensor pod in a restart loop.
func TestETCDTargetsRemainStableAcrossReconciles(t *testing.T) {
	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
		},
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				Key: "test-key",
			},
			Zone: instanav1.Name{
				Name: "test-zone",
			},
		},
	}

	ctx := context.Background()
	logger := zap.New()

	discoveredTargets := []string{"http://9.60.248.41:2381/metrics"}
	discoverETCD := func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
		return &DiscoveredETCDTargets{Targets: discoveredTargets, CAFound: false}, nil
	}

	// First reconcile: no Deployment exists yet, so the targets get applied.
	firstClient := &mocks.MockInstanaAgentClient{}
	firstClient.On("Get", mock.Anything, mock.Anything, mock.AnythingOfType("*v1.Deployment"), mock.Anything).
		Return(apierrors.NewNotFound(schema.GroupResource{}, ""))

	firstContext, err := CreateDeploymentContext(
		ctx,
		firstClient,
		agent,
		false,
		logger,
		discoverETCD,
	)
	require.NoError(t, err)

	firstDeployment := renderK8sSensorDeployment(t, agent, firstContext)
	assert.Equal(
		t,
		"http://9.60.248.41:2381/metrics",
		etcdTargetsEnvOf(firstDeployment),
		"first reconcile should set ETCD_TARGETS",
	)

	// Second reconcile: the Deployment now exists and discovery returns the same
	// targets, which is the case that used to strip the env var.
	secondClient := &mocks.MockInstanaAgentClient{}
	secondClient.On("Get", mock.Anything, mock.Anything, mock.AnythingOfType("*v1.Deployment"), mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			deployment := args.Get(2).(*appsv1.Deployment)
			*deployment = *firstDeployment
		})

	secondContext, err := CreateDeploymentContext(
		ctx,
		secondClient,
		agent,
		false,
		logger,
		discoverETCD,
	)
	require.NoError(t, err)

	secondDeployment := renderK8sSensorDeployment(t, agent, secondContext)
	assert.Equal(
		t,
		etcdTargetsEnvOf(firstDeployment),
		etcdTargetsEnvOf(secondDeployment),
		"ETCD_TARGETS must survive a reconcile where the discovered targets are unchanged",
	)

	// The pod spec must be byte for byte identical, otherwise the apply rolls the pod.
	assert.Equal(
		t,
		firstDeployment.Spec.Template.Spec.Containers[0].Env,
		secondDeployment.Spec.Template.Spec.Containers[0].Env,
		"an unchanged reconcile must not change the container env at all",
	)

	firstClient.AssertExpectations(t)
	secondClient.AssertExpectations(t)
}

// TestETCDTargetsUpdateWhenDiscoveryChanges makes sure the fix does not pin the env
// var: a genuine change in the discovered targets still rolls through.
func TestETCDTargetsUpdateWhenDiscoveryChanges(t *testing.T) {
	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
		},
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				Key: "test-key",
			},
			Zone: instanav1.Name{
				Name: "test-zone",
			},
		},
	}

	ctx := context.Background()
	logger := zap.New()

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
									Value: "http://9.60.248.41:2381/metrics",
								},
							},
						},
					},
				},
			},
		},
	}

	discoverETCD := func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
		return &DiscoveredETCDTargets{
			Targets: []string{"http://9.60.248.42:2381/metrics"},
			CAFound: false,
		}, nil
	}

	mockClient := &mocks.MockInstanaAgentClient{}
	mockClient.On("Get", mock.Anything, mock.Anything, mock.AnythingOfType("*v1.Deployment"), mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			deployment := args.Get(2).(*appsv1.Deployment)
			*deployment = *existingDeployment
		})

	deploymentContext, err := CreateDeploymentContext(
		ctx,
		mockClient,
		agent,
		false,
		logger,
		discoverETCD,
	)
	require.NoError(t, err)

	deployment := renderK8sSensorDeployment(t, agent, deploymentContext)
	assert.Equal(
		t,
		"http://9.60.248.42:2381/metrics",
		etcdTargetsEnvOf(deployment),
		"a changed discovery result should update ETCD_TARGETS",
	)
	mockClient.AssertExpectations(t)
}
