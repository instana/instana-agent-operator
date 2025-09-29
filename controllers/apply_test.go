package controllers

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/k8s-sensor/deployment"
)

// mockReconciler is a simplified version of InstanaAgentReconciler for testing
type mockReconciler struct {
	client               client.Client
	mockDiscoverETCDFunc func(ctx context.Context,
		agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error)
	mockCreateServiceCAConfigMapFunc func(ctx context.Context, agent *instanav1.InstanaAgent) error
}

func (r *mockReconciler) loggerFor(ctx context.Context, agent *instanav1.InstanaAgent) logr.Logger {
	return zap.New().WithValues("test", "test")
}

func (r *mockReconciler) DiscoverETCDEndpoints(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
	return r.mockDiscoverETCDFunc(ctx, agent)
}

func (r *mockReconciler) createServiceCAConfigMap(ctx context.Context, agent *instanav1.InstanaAgent) error {
	return r.mockCreateServiceCAConfigMapFunc(ctx, agent)
}

// Implement the createDeploymentContext method for our mock reconciler
func (r *mockReconciler) createDeploymentContext(
	ctx context.Context,
	agent *instanav1.InstanaAgent,
	isOpenShift bool,
) (*deployment.DeploymentContext, reconcileReturn) {
	log := r.loggerFor(ctx, agent)
	var deploymentContext *deployment.DeploymentContext

	// For OpenShift, create the service-CA ConfigMap
	if isOpenShift {
		if err := r.createServiceCAConfigMap(ctx, agent); err != nil {
			log.Error(err, "Failed to create service-CA ConfigMap")
			// Continue with reconciliation, don't fail the whole process
		} else {
			// Set up deployment context for OpenShift
			deploymentContext = &deployment.DeploymentContext{
				ETCDCASecretName: constants.ServiceCAConfigMapName,
			}
		}
	} else {
		// For vanilla Kubernetes, discover ETCD endpoints
		discoveredETCD, err := r.DiscoverETCDEndpoints(ctx, agent)
		if err != nil {
			log.Error(err, "Failed to discover ETCD endpoints")
			// Continue with reconciliation, don't fail the whole process
		} else if discoveredETCD != nil && len(discoveredETCD.Targets) > 0 {
			// Check if we need to update the Deployment with new ETCD targets
			existingDeployment := &appsv1.Deployment{}
			helperInstance := helpers.NewHelpers(agent)
			err := r.client.Get(ctx, client.ObjectKey{
				Namespace: agent.Namespace,
				Name:      helperInstance.K8sSensorResourcesName(),
			}, existingDeployment)

			if err == nil {
				// Check if the ETCD_TARGETS env var already exists with the same value
				currentTargets := ""
				for _, container := range existingDeployment.Spec.Template.Spec.Containers {
					if container.Name == constants.ContainerK8Sensor {
						for _, env := range container.Env {
							if env.Name == constants.EnvETCDTargets {
								currentTargets = env.Value
								break
							}
						}
						break
					}
				}

				// Sort targets to ensure consistent comparison
				sortedTargets := make([]string, len(discoveredETCD.Targets))
				copy(sortedTargets, discoveredETCD.Targets)
				sort.Strings(sortedTargets)
				newTargets := strings.Join(sortedTargets, ",")

				// Sort currentTargets for proper comparison
				if currentTargets != "" {
					currentTargetsList := strings.Split(currentTargets, ",")
					sort.Strings(currentTargetsList)
					currentTargets = strings.Join(currentTargetsList, ",")
				}

				if currentTargets == newTargets {
					log.Info("ETCD targets unchanged, skipping Deployment update")
					return nil, reconcileSuccess(ctrl.Result{})
				}
			}

			// Use sorted targets for consistency
			sortedTargets := make([]string, len(discoveredETCD.Targets))
			copy(sortedTargets, discoveredETCD.Targets)
			sort.Strings(sortedTargets)

			log.Info("Using discovered ETCD targets", "targets", sortedTargets)
			deploymentContext = &deployment.DeploymentContext{
				DiscoveredETCDTargets: sortedTargets,
			}
			if discoveredETCD.CAFound {
				deploymentContext.ETCDCASecretName = constants.ETCDCASecretName
			}
		}
	}

	return deploymentContext, reconcileContinue()
}

func TestCreateDeploymentContext_OpenShift(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = instanav1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
		},
	}

	// Create a fake client
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Create mock reconciler
	reconciler := &mockReconciler{
		client: fakeClient,
		mockCreateServiceCAConfigMapFunc: func(ctx context.Context, agent *instanav1.InstanaAgent) error {
			return nil
		},
		mockDiscoverETCDFunc: func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
			return nil, nil
		},
	}

	// Test
	ctx := context.Background()
	deploymentContext, result := reconciler.createDeploymentContext(ctx, agent, true)

	// Verify
	require.False(t, result.suppliesReconcileResult(), "Should not supply reconcile result")
	require.NotNil(t, deploymentContext, "Deployment context should not be nil for OpenShift")
	assert.Equal(t, constants.ServiceCAConfigMapName, deploymentContext.ETCDCASecretName, "ETCDCASecretName should be set to ServiceCAConfigMapName")
}

func TestCreateDeploymentContext_VanillaK8s_NoETCD(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = instanav1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
		},
	}

	// Create a fake client
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Create mock reconciler
	reconciler := &mockReconciler{
		client: fakeClient,
		mockCreateServiceCAConfigMapFunc: func(ctx context.Context, agent *instanav1.InstanaAgent) error {
			return nil
		},
		mockDiscoverETCDFunc: func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
			return nil, nil
		},
	}

	// Test
	ctx := context.Background()
	deploymentContext, result := reconciler.createDeploymentContext(ctx, agent, false)

	// Verify
	require.False(t, result.suppliesReconcileResult(), "Should not supply reconcile result")
	assert.Nil(t, deploymentContext, "Deployment context should be nil when no ETCD is discovered")
}

func TestCreateDeploymentContext_VanillaK8s_WithETCD(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = instanav1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)

	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
		},
	}

	// Create a fake client
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Create mock reconciler
	reconciler := &mockReconciler{
		client: fakeClient,
		mockCreateServiceCAConfigMapFunc: func(ctx context.Context, agent *instanav1.InstanaAgent) error {
			return nil
		},
		mockDiscoverETCDFunc: func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
			return &DiscoveredETCDTargets{
				Targets: []string{"https://etcd-1:2379/metrics", "https://etcd-2:2379/metrics"},
				CAFound: true,
			}, nil
		},
	}

	// Test
	ctx := context.Background()
	deploymentContext, result := reconciler.createDeploymentContext(ctx, agent, false)

	// Verify
	require.False(t, result.suppliesReconcileResult(), "Should not supply reconcile result")
	require.NotNil(t, deploymentContext, "Deployment context should not be nil when ETCD is discovered")
	assert.Equal(t, []string{"https://etcd-1:2379/metrics", "https://etcd-2:2379/metrics"}, deploymentContext.DiscoveredETCDTargets, "DiscoveredETCDTargets should be set")
	assert.Equal(
		t,
		constants.ETCDCASecretName,
		deploymentContext.ETCDCASecretName,
		"ETCDCASecretName should be set to etcd-ca",
	)
}

func TestCreateDeploymentContext_VanillaK8s_UnchangedTargets(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = instanav1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)

	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
		},
	}

	// Create a deployment with existing ETCD targets
	helperInstance := helpers.NewHelpers(agent)
	existingDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      helperInstance.K8sSensorResourcesName(),
			Namespace: agent.Namespace,
		},
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

	// Create a fake client with the existing deployment
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(existingDeployment).
		Build()

	// Create mock reconciler
	reconciler := &mockReconciler{
		client: fakeClient,
		mockCreateServiceCAConfigMapFunc: func(ctx context.Context, agent *instanav1.InstanaAgent) error {
			return nil
		},
		mockDiscoverETCDFunc: func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
			return &DiscoveredETCDTargets{
				Targets: []string{"https://etcd-2:2379/metrics", "https://etcd-1:2379/metrics"}, // Order doesn't matter, they'll be sorted
				CAFound: true,
			}, nil
		},
	}

	// Test
	ctx := context.Background()
	deploymentContext, result := reconciler.createDeploymentContext(ctx, agent, false)

	// Verify
	require.True(t, result.suppliesReconcileResult(), "Should supply reconcile result when targets are unchanged")
	assert.Nil(t, deploymentContext, "Deployment context should be nil when targets are unchanged")
}

func TestCreateDeploymentContext_VanillaK8s_DifferentOrderTargets(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = instanav1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)

	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
		},
	}

	// Create a deployment with existing ETCD targets in a specific order
	helperInstance := helpers.NewHelpers(agent)
	existingDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      helperInstance.K8sSensorResourcesName(),
			Namespace: agent.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: constants.ContainerK8Sensor,
							Env: []corev1.EnvVar{
								{
									Name:  constants.EnvETCDTargets,
									Value: "https://etcd-2:2379/metrics,https://etcd-1:2379/metrics,https://etcd-3:2379/metrics", // Unsorted order
								},
							},
						},
					},
				},
			},
		},
	}

	// Create a fake client with the existing deployment
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(existingDeployment).
		Build()

	// Create mock reconciler
	reconciler := &mockReconciler{
		client: fakeClient,
		mockCreateServiceCAConfigMapFunc: func(ctx context.Context, agent *instanav1.InstanaAgent) error {
			return nil
		},
		mockDiscoverETCDFunc: func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
			return &DiscoveredETCDTargets{
				// Different order than in the deployment, but same targets
				Targets: []string{
					"https://etcd-1:2379/metrics",
					"https://etcd-3:2379/metrics",
					"https://etcd-2:2379/metrics",
				},
				CAFound: true,
			}, nil
		},
	}

	// Test
	ctx := context.Background()
	deploymentContext, result := reconciler.createDeploymentContext(ctx, agent, false)

	// Verify
	require.True(t,
		result.suppliesReconcileResult(),
		"Should supply reconcile result when targets are unchanged (just in different order)")
	assert.Nil(t, deploymentContext, "Deployment context should be nil when targets are unchanged")
}
