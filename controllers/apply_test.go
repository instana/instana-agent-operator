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
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/internal/mocks"
	instanaclient "github.com/instana/instana-agent-operator/pkg/k8s/client"
	backends "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/backends"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/namespaces"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/operator_utils"
	"github.com/instana/instana-agent-operator/pkg/pointer"
	"github.com/instana/instana-agent-operator/pkg/result"
)

type mockOperatorUtils struct {
	applyAllCalled bool
	builders       []builder.ObjectBuilder
}

func (m *mockOperatorUtils) ClusterIsOpenShift() (bool, error) {
	return false, nil
}

func (m *mockOperatorUtils) ApplyAll(builders ...builder.ObjectBuilder) error {
	m.applyAllCalled = true
	m.builders = builders
	return nil
}

func (m *mockOperatorUtils) DeleteAll() error {
	return nil
}

var _ operator_utils.OperatorUtils = (*mockOperatorUtils)(nil)

func TestCreateDeploymentContext_SimplifiedTests(t *testing.T) {
	t.Setenv("POD_NAMESPACE", "anything")
	_ = os.Unsetenv("POD_NAMESPACE")
	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-agent",
			// From now on, only those namespaces get auto ETCD discovery on OpenShift, that match the operator's namespace and `instana-agent` is the assumed default.
			Namespace: "instana-agent",
		},
	}

	ctx := context.Background()
	logger := zap.New()

	t.Run("OpenShift discovers and copies ETCD resources", func(t *testing.T) {
		mockClient := &mocks.MockInstanaAgentClient{}

		// Mock the Apply method for copying ETCD ConfigMap and Secret
		mockClient.On("Apply", mock.Anything, mock.AnythingOfType("*v1.ConfigMap"), mock.Anything).
			Return(result.OfSuccess[client.Object](nil))
		mockClient.On("Apply", mock.Anything, mock.AnythingOfType("*v1.Secret"), mock.Anything).
			Return(result.OfSuccess[client.Object](nil))

		// Mock Get calls for ETCD resource checks with valid data
		mockClient.On(
			"Get",
			mock.Anything,
			mock.Anything,
			mock.AnythingOfType("*v1.ConfigMap"),
			mock.Anything,
		).Run(func(args mock.Arguments) {
			cm := args.Get(2).(*corev1.ConfigMap)
			cm.Data = map[string]string{
				"ca-bundle.crt": "test-ca-cert-data",
			}
			cm.ResourceVersion = "12345"
		}).Return(nil)
		mockClient.On(
			"Get",
			mock.Anything,
			mock.Anything,
			mock.AnythingOfType("*v1.Secret"),
			mock.Anything,
		).Run(func(args mock.Arguments) {
			secret := args.Get(2).(*corev1.Secret)
			secret.Data = map[string][]byte{
				"tls.crt": []byte("test-cert-data"),
				"tls.key": []byte("test-key-data"),
			}
			secret.ResourceVersion = "67890"
		}).Return(nil)

		// Mock ETCD discover function (won't be called for OpenShift)
		mockDiscoverETCD := func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
			return nil, nil
		}

		deploymentContext, err := CreateDeploymentContext(
			ctx,
			mockClient,
			agent,
			true,
			logger,
			mockDiscoverETCD,
		)

		require.NoError(t, err)
		require.NotNil(t, deploymentContext)
		assert.True(
			t,
			deploymentContext.OpenShiftETCDResourcesExist,
			"ETCD resources should exist when Get calls succeed",
		)
		mockClient.AssertExpectations(t)
	})

	t.Run("Vanilla K8s with no ETCD returns nil", func(t *testing.T) {
		mockClient := &mocks.MockInstanaAgentClient{}

		// Mock ETCD discover function that returns nil
		mockDiscoverETCD := func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
			return nil, nil
		}

		deploymentContext, err := CreateDeploymentContext(
			ctx,
			mockClient,
			agent,
			false,
			logger,
			mockDiscoverETCD,
		)

		require.NoError(t, err)
		assert.Nil(t, deploymentContext)
		mockClient.AssertExpectations(t)
	})

	t.Run("Vanilla K8s with a discovered CA creates deployment context", func(t *testing.T) {
		mockClient := &mocks.MockInstanaAgentClient{}

		// Mock ETCD discover function that finds etcd and a CA for it
		mockDiscoverETCD := func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
			return &DiscoveredETCDTargets{
				Targets: []string{"https://etcd-1:2379/metrics", "https://etcd-2:2379/metrics"},
				CAFound: true,
			}, nil
		}

		deploymentContext, err := CreateDeploymentContext(
			ctx,
			mockClient,
			agent,
			false,
			logger,
			mockDiscoverETCD,
		)

		require.NoError(t, err)
		require.NotNil(t, deploymentContext)
		assert.Equal(t, constants.ETCDCASecretName, deploymentContext.ETCDCASecretName)
		mockClient.AssertExpectations(t)
	})

	t.Run("Vanilla K8s without a CA returns nil", func(t *testing.T) {
		mockClient := &mocks.MockInstanaAgentClient{}

		// etcd is there but no CA secret, so there is nothing for the operator to add
		mockDiscoverETCD := func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
			return &DiscoveredETCDTargets{
				Targets: []string{"https://etcd-1:2379/metrics"},
				CAFound: false,
			}, nil
		}

		deploymentContext, err := CreateDeploymentContext(
			ctx,
			mockClient,
			agent,
			false,
			logger,
			mockDiscoverETCD,
		)

		require.NoError(t, err)
		assert.Nil(t, deploymentContext)
		mockClient.AssertExpectations(t)
	})

	t.Run("Context does not depend on the live Deployment", func(t *testing.T) {
		// The context is derived from discovery alone, so two consecutive reconciles
		// produce the same thing and the rendered Deployment never churns. The mock
		// client has no expectations at all, which asserts that the live Deployment is
		// never read.
		mockClient := &mocks.MockInstanaAgentClient{}

		mockDiscoverETCD := func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
			return &DiscoveredETCDTargets{
				Targets: []string{"https://etcd-1:2379/metrics"},
				CAFound: true,
			}, nil
		}

		first, err := CreateDeploymentContext(
			ctx,
			mockClient,
			agent,
			false,
			logger,
			mockDiscoverETCD,
		)
		require.NoError(t, err)
		second, err := CreateDeploymentContext(
			ctx,
			mockClient,
			agent,
			false,
			logger,
			mockDiscoverETCD,
		)
		require.NoError(t, err)

		assert.Equal(t, first, second, "consecutive reconciles must produce the same context")
		mockClient.AssertExpectations(t)
	})

	t.Run("Error handling in ETCD discovery", func(t *testing.T) {
		mockClient := &mocks.MockInstanaAgentClient{}

		// Mock ETCD discover function that returns error
		mockDiscoverETCD := func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
			return nil, assert.AnError
		}

		deploymentContext, err := CreateDeploymentContext(
			ctx,
			mockClient,
			agent,
			false,
			logger,
			mockDiscoverETCD,
		)

		require.NoError(t, err) // Function continues on error
		assert.Nil(t, deploymentContext)
		mockClient.AssertExpectations(t)
	})

	t.Run(
		"OpenShift namespace validation - agent in same namespace as operator",
		func(t *testing.T) {
			t.Setenv("POD_NAMESPACE", "instana-agent")

			agentInSameNamespace := &instanav1.InstanaAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "instana-agent",
				},
			}

			mockClient := &mocks.MockInstanaAgentClient{}

			// Mock the Apply method for copying ETCD ConfigMap and Secret
			mockClient.On("Apply", mock.Anything, mock.AnythingOfType("*v1.ConfigMap"), mock.Anything).
				Return(result.OfSuccess[client.Object](nil))
			mockClient.On("Apply", mock.Anything, mock.AnythingOfType("*v1.Secret"), mock.Anything).
				Return(result.OfSuccess[client.Object](nil))

			// Mock Get calls for ETCD resource checks with valid data
			mockClient.On(
				"Get",
				mock.Anything,
				mock.Anything,
				mock.AnythingOfType("*v1.ConfigMap"),
				mock.Anything,
			).Run(func(args mock.Arguments) {
				cm := args.Get(2).(*corev1.ConfigMap)
				cm.Data = map[string]string{
					"ca-bundle.crt": "test-ca-cert-data",
				}
				cm.ResourceVersion = "12345"
			}).Return(nil)
			mockClient.On(
				"Get",
				mock.Anything,
				mock.Anything,
				mock.AnythingOfType("*v1.Secret"),
				mock.Anything,
			).Run(func(args mock.Arguments) {
				secret := args.Get(2).(*corev1.Secret)
				secret.Data = map[string][]byte{
					"tls.crt": []byte("test-cert-data"),
					"tls.key": []byte("test-key-data"),
				}
				secret.ResourceVersion = "67890"
			}).Return(nil)

			mockDiscoverETCD := func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
				return nil, nil
			}

			deploymentContext, err := CreateDeploymentContext(
				ctx,
				mockClient,
				agentInSameNamespace,
				true,
				logger,
				mockDiscoverETCD,
			)

			require.NoError(t, err)
			require.NotNil(t, deploymentContext)
			assert.True(
				t,
				deploymentContext.OpenShiftETCDResourcesExist,
				"ETCD resources should exist when agent is in same namespace as operator",
			)
			mockClient.AssertExpectations(t)
		},
	)

	t.Run(
		"OpenShift namespace validation - agent in different namespace skips ETCD",
		func(t *testing.T) {
			t.Setenv("POD_NAMESPACE", "instana-agent")

			agentInDifferentNamespace := &instanav1.InstanaAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "other-namespace",
				},
			}

			mockClient := &mocks.MockInstanaAgentClient{}

			mockDiscoverETCD := func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
				t.Fatal("discoverETCD should not be called when namespaces differ")
				return nil, nil
			}

			deploymentContext, err := CreateDeploymentContext(
				ctx,
				mockClient,
				agentInDifferentNamespace,
				true,
				logger,
				mockDiscoverETCD,
			)

			require.NoError(t, err)
			assert.Nil(
				t,
				deploymentContext,
				"Should return nil when agent namespace differs from operator namespace",
			)
			mockClient.AssertExpectations(t)
		},
	)

	t.Run(
		"OpenShift namespace validation - POD_NAMESPACE not set uses default",
		func(t *testing.T) {
			// Ensure POD_NAMESPACE is not set
			t.Setenv("POD_NAMESPACE", "anything")
			_ = os.Unsetenv("POD_NAMESPACE")

			agentInDefaultNamespace := &instanav1.InstanaAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "instana-agent",
				},
			}

			mockClient := &mocks.MockInstanaAgentClient{}

			// Mock the Apply method for copying ETCD ConfigMap and Secret
			mockClient.On("Apply", mock.Anything, mock.AnythingOfType("*v1.ConfigMap"), mock.Anything).
				Return(result.OfSuccess[client.Object](nil))
			mockClient.On("Apply", mock.Anything, mock.AnythingOfType("*v1.Secret"), mock.Anything).
				Return(result.OfSuccess[client.Object](nil))

			// Mock Get calls for ETCD resource checks with valid data
			mockClient.On(
				"Get",
				mock.Anything,
				mock.Anything,
				mock.AnythingOfType("*v1.ConfigMap"),
				mock.Anything,
			).Run(func(args mock.Arguments) {
				cm := args.Get(2).(*corev1.ConfigMap)
				cm.Data = map[string]string{
					"ca-bundle.crt": "test-ca-cert-data",
				}
				cm.ResourceVersion = "12345"
			}).Return(nil)
			mockClient.On(
				"Get",
				mock.Anything,
				mock.Anything,
				mock.AnythingOfType("*v1.Secret"),
				mock.Anything,
			).Run(func(args mock.Arguments) {
				secret := args.Get(2).(*corev1.Secret)
				secret.Data = map[string][]byte{
					"tls.crt": []byte("test-cert-data"),
					"tls.key": []byte("test-key-data"),
				}
				secret.ResourceVersion = "67890"
			}).Return(nil)

			mockDiscoverETCD := func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
				return nil, nil
			}

			deploymentContext, err := CreateDeploymentContext(
				ctx,
				mockClient,
				agentInDefaultNamespace,
				true,
				logger,
				mockDiscoverETCD,
			)

			require.NoError(t, err)
			require.NotNil(t, deploymentContext)
			assert.True(
				t,
				deploymentContext.OpenShiftETCDResourcesExist,
				"ETCD resources should exist when POD_NAMESPACE not set and agent in default namespace",
			)
			mockClient.AssertExpectations(t)
		},
	)

	t.Run(
		"OpenShift namespace validation - POD_NAMESPACE not set with different agent namespace",
		func(t *testing.T) {
			// Ensure POD_NAMESPACE is not set
			t.Setenv("POD_NAMESPACE", "")

			agentInOtherNamespace := &instanav1.InstanaAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "other-namespace",
				},
			}

			mockClient := &mocks.MockInstanaAgentClient{}

			mockDiscoverETCD := func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
				t.Fatal("discoverETCD should not be called when namespaces differ")
				return nil, nil
			}

			deploymentContext, err := CreateDeploymentContext(
				ctx,
				mockClient,
				agentInOtherNamespace,
				true,
				logger,
				mockDiscoverETCD,
			)

			require.NoError(t, err)
			assert.Nil(
				t,
				deploymentContext,
				"Should return nil when agent namespace differs from default operator namespace",
			)
			mockClient.AssertExpectations(t)
		},
	)
}

func TestGetDaemonSetBuildersReturnsFailureForZoneDaemonSetReadError(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	getErr := errors.New("apiserver temporary failure")
	baseClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	reconciler := &InstanaAgentReconciler{
		client: &getErrorInstanaAgentClient{
			InstanaAgentClient: instanaclient.NewInstanaAgentClient(baseClient),
			err:                getErr,
		},
	}

	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "agent",
			Namespace: "instana-agent",
		},
		Spec: instanav1.InstanaAgentSpec{
			Zones: []instanav1.Zone{
				{
					Name: instanav1.Name{Name: "zone-a"},
				},
			},
		},
	}

	builders, res := getDaemonSetBuilders(
		context.Background(),
		reconciler,
		agent,
		true,
		false,
		&mocks.MockAgentStatusManager{},
	)

	assert.Nil(t, builders)
	assert.True(t, res.suppliesReconcileResult())
	_, err := res.reconcileResult()
	assert.ErrorIs(t, err, getErr)
}

func TestApplyResourcesReturnsFailureAndSkipsApplyAllOnZoneDaemonSetReadError(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	getErr := errors.New("daemonset read failed")
	baseClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	reconciler := &InstanaAgentReconciler{
		client: &getErrorInstanaAgentClient{
			InstanaAgentClient: instanaclient.NewInstanaAgentClient(baseClient),
			err:                getErr,
		},
	}

	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "agent",
			Namespace: "instana-agent",
		},
		Spec: instanav1.InstanaAgentSpec{
			Zones: []instanav1.Zone{
				{
					Name: instanav1.Name{Name: "zone-a"},
				},
			},
		},
	}

	operatorUtilsMock := &mockOperatorUtils{}
	res := reconciler.applyResources(
		context.Background(),
		agent,
		true,
		false,
		operatorUtilsMock,
		&mocks.MockAgentStatusManager{},
		&corev1.Secret{},
		nil,
		namespaces.NamespacesDetails{},
	)

	assert.True(t, res.suppliesReconcileResult())
	_, err := res.reconcileResult()
	assert.ErrorIs(t, err, getErr)
	assert.False(t, operatorUtilsMock.applyAllCalled)
}

// Namespaced objects owned by the agent CR must live in the CR's namespace, otherwise
// Kubernetes treats the owner reference as invalid and garbage collects them, which the
// operator then recreates on every reconcile.
func TestApplyResourcesOnlyBuildsNamespacedObjectsInTheAgentNamespace(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	baseClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	reconciler := &InstanaAgentReconciler{
		client: instanaclient.NewInstanaAgentClient(baseClient),
	}

	// A fully populated spec, so that the conditionally built objects (agent DaemonSet,
	// k8sensor Deployment, PodDisruptionBudget, secrets) are actually emitted and get
	// checked, rather than being skipped as empty Optionals.
	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "instana-agent",
			Namespace: "base-mon",
		},
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				Key:          "kjfdoisjoifjdsoijdf",
				EndpointHost: "ingress.instana.io",
				EndpointPort: "443",
			},
			Cluster: instanav1.Name{Name: "oiweisjdfoi"},
			K8sSensor: instanav1.K8sSpec{
				PodDisruptionBudget: instanav1.Enabled{Enabled: pointer.To(true)},
			},
		},
	}
	agent.Default()

	k8SensorBackends := []backends.K8SensorBackend{
		*backends.NewK8SensorBackend("", agent.Spec.Agent.Key, "", "ingress.instana.io", "443"),
	}

	// Asserted rather than permissive, so that a builder dropping out of the set below
	// fails here instead of shrinking what the namespace check covers.
	statusManager := &mocks.MockAgentStatusManager{}
	defer statusManager.AssertExpectations(t)
	statusManager.On("AddAgentDaemonset", mock.Anything).Return().Once()
	statusManager.On("SetAgentSecretConfig", mock.Anything).Return().Once()
	statusManager.On("SetAgentNamespacesConfigMap", mock.Anything).Return().Once()
	statusManager.On("SetK8sSensorDeployment", mock.Anything).Return().Once()

	operatorUtilsMock := &mockOperatorUtils{}
	res := reconciler.applyResources(
		context.Background(),
		agent,
		true,
		false,
		operatorUtilsMock,
		statusManager,
		&corev1.Secret{},
		k8SensorBackends,
		namespaces.NamespacesDetails{},
	)

	assert.False(t, res.suppliesReconcileResult())
	require.True(t, operatorUtilsMock.applyAllCalled)

	checkedKinds := make(map[string]int, len(operatorUtilsMock.builders))
	checked := 0

	for _, objectBuilder := range operatorUtilsMock.builders {
		if !objectBuilder.IsNamespaced() {
			continue
		}

		// The mock ApplyAll only records the builders, so this is the one and only Build
		// call, which is what makes the status manager expectations above exact.
		object := objectBuilder.Build()
		if !object.IsPresent() {
			continue
		}

		obj := object.Get()
		checkedKinds[obj.GetObjectKind().GroupVersionKind().Kind]++
		checked++

		assert.Equal(
			t,
			agent.Namespace,
			obj.GetNamespace(),
			"%s %s is built outside of the agent namespace",
			obj.GetObjectKind().GroupVersionKind().Kind,
			obj.GetName(),
		)
	}

	// Guards against the loop above quietly degrading to checking almost nothing if a
	// builder stops emitting an object for this spec. The per kind counts are exact
	// rather than non-zero, because most kinds have more than one builder and a
	// non-zero check would not notice one of them dropping out.
	assert.Equal(
		t,
		map[string]int{
			"ConfigMap":           2,
			"Secret":              2, // pragma: allowlist secret
			"ServiceAccount":      2,
			"Service":             2,
			"DaemonSet":           1,
			"Deployment":          1,
			"PodDisruptionBudget": 1,
		},
		checkedKinds,
	)
	assert.Equal(t, 11, checked, "unexpected number of namespaced objects checked")
}
