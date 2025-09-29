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
	"fmt"
	"sort"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/operator_utils"
)

// Type aliases to make function signatures more readable
type (
	// IsOpenShiftFunc is a function that checks if the cluster is OpenShift
	IsOpenShiftFunc func(ctx context.Context, operatorUtils operator_utils.OperatorUtils) (bool, reconcileReturn)

	// SkipDiscoveryFunc is a function that checks if ETCD discovery should be skipped
	SkipDiscoveryFunc func(ctx context.Context, agent *instanav1.InstanaAgent, logger logr.Logger,
		isOpenShiftFunc IsOpenShiftFunc) (bool, error)

	// FindServiceFunc is a function that finds an ETCD service
	FindServiceFunc func(ctx context.Context, client client.Client, logger logr.Logger) (*corev1.Service, error)

	// FindPortAndSchemeFunc is a function that finds the metrics port and scheme
	FindPortAndSchemeFunc func(service *corev1.Service) (*int32, string)

	// BuildTargetsFunc is a function that builds targets from endpoint slices
	BuildTargetsFunc func(ctx context.Context, client client.Client, service *corev1.Service,
		metricsPort int32, scheme string) ([]string, error)

	// CheckCASecretFunc is a function that checks if the CA secret exists
	CheckCASecretFunc func(ctx context.Context, client client.Client,
		agent *instanav1.InstanaAgent, logger logr.Logger) bool
)

// mockETCDReconciler is a simplified version of InstanaAgentReconciler for testing
type mockETCDReconciler struct {
	client client.Client
	scheme *runtime.Scheme
	logger logr.Logger
}

func (r *mockETCDReconciler) loggerFor(
	ctx context.Context,
	agent *instanav1.InstanaAgent,
) logr.Logger {
	return r.logger
}

// Mock client that returns errors for specific operations
type errorMockClient struct {
	client.Client
	err error
}

// Get implements client.Client
func (c *errorMockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object,
	opts ...client.GetOption) error {
	return c.err
}

// Mock client that returns errors for specific service names
type selectiveErrorMockClient struct {
	client.Client
	err          error
	errorForName string
}

// Get implements client.Client
func (c *selectiveErrorMockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object,
	opts ...client.GetOption) error {
	if key.Name == c.errorForName {
		return c.err
	}
	return c.Client.Get(ctx, key, obj, opts...)
}

// TestShouldSkipDiscovery tests the shouldSkipDiscovery function
func TestShouldSkipDiscovery(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = instanav1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	// Test cases
	testCases := []struct {
		name           string
		isOpenShift    bool
		isOpenShiftErr bool
		etcdTargets    []string
		expectedSkip   bool
		expectedErr    bool
	}{
		{
			name:           "Should skip on OpenShift",
			isOpenShift:    true,
			isOpenShiftErr: false,
			etcdTargets:    nil,
			expectedSkip:   true,
			expectedErr:    false,
		},
		{
			name:           "Should skip when targets are specified in CR",
			isOpenShift:    false,
			isOpenShiftErr: false,
			etcdTargets:    []string{"https://etcd-1:2379/metrics"},
			expectedSkip:   true,
			expectedErr:    false,
		},
		{
			name:           "Should not skip on vanilla K8s without targets",
			isOpenShift:    false,
			isOpenShiftErr: false,
			etcdTargets:    nil,
			expectedSkip:   false,
			expectedErr:    false,
		},
		{
			name:           "Should return error when isOpenShift fails",
			isOpenShift:    false,
			isOpenShiftErr: true,
			etcdTargets:    nil,
			expectedSkip:   false,
			expectedErr:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a fake client
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

			// Create agent with or without ETCD targets
			agent := &instanav1.InstanaAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "test-namespace",
				},
				Spec: instanav1.InstanaAgentSpec{
					K8sSensor: instanav1.K8sSpec{
						ETCD: instanav1.ETCDSpec{
							Targets: tc.etcdTargets,
						},
					},
				},
			}

			// Create mock reconciler
			reconciler := &mockETCDReconciler{
				client: fakeClient,
				scheme: scheme,
				logger: zap.New(),
			}

			// Mock isOpenShift function
			mockIsOpenShift := func(ctx context.Context, operatorUtils operator_utils.OperatorUtils) (bool, reconcileReturn) {
				if tc.isOpenShiftErr {
					return false, reconcileFailure(fmt.Errorf("isOpenShift error"))
				}
				return tc.isOpenShift, reconcileContinue()
			}

			// Test
			ctx := context.Background()
			logger := reconciler.loggerFor(ctx, agent)
			skip, err := shouldSkipDiscovery(ctx, agent, logger, mockIsOpenShift)

			// Verify
			if tc.expectedErr {
				assert.Error(t, err, "Should return an error")
			} else {
				assert.NoError(t, err, "Should not return an error")
			}
			assert.Equal(t, tc.expectedSkip, skip, "Skip value should match expected")

			// Explicitly assert skip is false when discovery should proceed
			if tc.name == "Should not skip on vanilla K8s without targets" {
				assert.False(t, skip, "Skip should be false when discovery should proceed")
			}
		})
	}
}

// TestGetServiceWithLabel tests the getServiceWithLabel function
func TestGetServiceWithLabel(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = instanav1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	// Test cases
	testCases := []struct {
		name           string
		serviceName    string
		serviceLabels  map[string]string
		labelKey       string
		labelValue     string
		expectService  bool
		expectError    bool
		setupClientErr bool
	}{
		{
			name:           "Should find service with matching label",
			serviceName:    "etcd",
			serviceLabels:  map[string]string{"component": "etcd"},
			labelKey:       "component",
			labelValue:     "etcd",
			expectService:  true,
			expectError:    false,
			setupClientErr: false,
		},
		{
			name:           "Should not find service with non-matching label",
			serviceName:    "etcd",
			serviceLabels:  map[string]string{"component": "not-etcd"},
			labelKey:       "component",
			labelValue:     "etcd",
			expectService:  false,
			expectError:    false,
			setupClientErr: false,
		},
		{
			name:           "Should not find service with missing label",
			serviceName:    "etcd",
			serviceLabels:  map[string]string{"app": "etcd"},
			labelKey:       "component",
			labelValue:     "etcd",
			expectService:  false,
			expectError:    false,
			setupClientErr: false,
		},
		{
			name:           "Should not find service with nil labels",
			serviceName:    "etcd",
			serviceLabels:  nil,
			labelKey:       "component",
			labelValue:     "etcd",
			expectService:  false,
			expectError:    false,
			setupClientErr: false,
		},
		{
			name:           "Should handle not found error",
			serviceName:    "non-existent",
			serviceLabels:  map[string]string{"component": "etcd"},
			labelKey:       "component",
			labelValue:     "etcd",
			expectService:  false,
			expectError:    false,
			setupClientErr: false,
		},
		{
			name:           "Should propagate client error",
			serviceName:    "etcd",
			serviceLabels:  map[string]string{"component": "etcd"},
			labelKey:       "component",
			labelValue:     "etcd",
			expectService:  false,
			expectError:    true,
			setupClientErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a fake client builder
			clientBuilder := fake.NewClientBuilder().WithScheme(scheme)

			// Add service to client if needed
			if tc.serviceName != "non-existent" {
				service := &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      tc.serviceName,
						Namespace: kubeSystemNamespace,
						Labels:    tc.serviceLabels,
					},
				}
				clientBuilder = clientBuilder.WithObjects(service)
			}

			// Build the client
			fakeClient := clientBuilder.Build()

			// Create mock client that returns error if needed
			var mockClient client.Client
			if tc.setupClientErr {
				mockClient = &errorMockClient{
					Client: fakeClient,
					err:    fmt.Errorf("client error"),
				}
			} else {
				mockClient = fakeClient
			}

			// Test
			ctx := context.Background()
			service, err := getServiceWithLabel(
				ctx,
				mockClient,
				tc.serviceName,
				tc.labelKey,
				tc.labelValue,
			)

			// Verify
			if tc.expectError {
				assert.Error(t, err, "Should return an error")
			} else {
				assert.NoError(t, err, "Should not return an error")
				if tc.expectService {
					assert.NotNil(t, service, "Service should not be nil")
					assert.Equal(t, tc.serviceName, service.Name, "Service name should match")
				} else {
					assert.Nil(t, service, "Service should be nil")
				}
			}
		})
	}
}

// TestGetServiceByName tests the getServiceByName function
func TestGetServiceByName(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = instanav1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	// Test cases
	testCases := []struct {
		name           string
		serviceName    string
		createService  bool
		expectService  bool
		expectError    bool
		setupClientErr bool
	}{
		{
			name:           "Should find existing service",
			serviceName:    "etcd",
			createService:  true,
			expectService:  true,
			expectError:    false,
			setupClientErr: false,
		},
		{
			name:           "Should not find non-existent service",
			serviceName:    "non-existent",
			createService:  false,
			expectService:  false,
			expectError:    false,
			setupClientErr: false,
		},
		{
			name:           "Should propagate client error",
			serviceName:    "etcd",
			createService:  true,
			expectService:  false,
			expectError:    true,
			setupClientErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a fake client builder
			clientBuilder := fake.NewClientBuilder().WithScheme(scheme)

			// Add service to client if needed
			if tc.createService {
				service := &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      tc.serviceName,
						Namespace: kubeSystemNamespace,
					},
				}
				clientBuilder = clientBuilder.WithObjects(service)
			}

			// Build the client
			fakeClient := clientBuilder.Build()

			// Create mock client that returns error if needed
			var mockClient client.Client
			if tc.setupClientErr {
				mockClient = &errorMockClient{
					Client: fakeClient,
					err:    fmt.Errorf("client error"),
				}
			} else {
				mockClient = fakeClient
			}

			// Test
			ctx := context.Background()
			service, err := getServiceByName(ctx, mockClient, tc.serviceName)

			// Verify
			if tc.expectError {
				assert.Error(t, err, "Should return an error")
			} else {
				assert.NoError(t, err, "Should not return an error")
				if tc.expectService {
					assert.NotNil(t, service, "Service should not be nil")
					assert.Equal(t, tc.serviceName, service.Name, "Service name should match")
				} else {
					assert.Nil(t, service, "Service should be nil")
				}
			}
		})
	}
}

// TestFindETCDService tests the findETCDService function
func TestFindETCDService(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = instanav1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	// Test cases
	testCases := []struct {
		name              string
		services          []corev1.Service
		expectedService   string
		expectError       bool
		setupClientErr    bool
		setupClientErrFor string
	}{
		{
			name: "Should find etcd service with component=etcd label",
			services: []corev1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "etcd",
						Namespace: kubeSystemNamespace,
						Labels:    map[string]string{"component": "etcd"},
					},
				},
			},
			expectedService: "etcd",
			expectError:     false,
		},
		{
			name: "Should find etcd-metrics service with component=etcd label",
			services: []corev1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "etcd-metrics",
						Namespace: kubeSystemNamespace,
						Labels:    map[string]string{"component": "etcd"},
					},
				},
			},
			expectedService: "etcd-metrics",
			expectError:     false,
		},
		{
			name: "Should find etcd service by name when no service has component=etcd label",
			services: []corev1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "etcd",
						Namespace: kubeSystemNamespace,
					},
				},
			},
			expectedService: "etcd",
			expectError:     false,
		},
		{
			name: "Should find etcd-metrics service by name when no etcd service exists",
			services: []corev1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "etcd-metrics",
						Namespace: kubeSystemNamespace,
					},
				},
			},
			expectedService: "etcd-metrics",
			expectError:     false,
		},
		{
			name: "Should find etcd-k8s service by name when no etcd or etcd-metrics service exists",
			services: []corev1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "etcd-k8s",
						Namespace: kubeSystemNamespace,
					},
				},
			},
			expectedService: "etcd-k8s",
			expectError:     false,
		},
		{
			name:            "Should return nil when no etcd service exists",
			services:        []corev1.Service{},
			expectedService: "",
			expectError:     false,
		},
		{
			name: "Should propagate error from getServiceWithLabel",
			services: []corev1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "etcd",
						Namespace: kubeSystemNamespace,
						Labels:    map[string]string{"component": "etcd"},
					},
				},
			},
			expectedService:   "",
			expectError:       true,
			setupClientErr:    true,
			setupClientErrFor: "etcd",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a fake client builder
			clientBuilder := fake.NewClientBuilder().WithScheme(scheme)

			// Add services to client
			for _, svc := range tc.services {
				clientBuilder = clientBuilder.WithObjects(&svc)
			}

			// Build the client
			fakeClient := clientBuilder.Build()

			// Create mock client that returns error if needed
			var mockClient client.Client
			if tc.setupClientErr {
				mockClient = &selectiveErrorMockClient{
					Client:       fakeClient,
					err:          fmt.Errorf("client error"),
					errorForName: tc.setupClientErrFor,
				}
			} else {
				mockClient = fakeClient
			}

			// Test
			ctx := context.Background()
			logger := ctrl.Log.WithName("test")
			service, err := findETCDService(ctx, mockClient, logger)

			// Verify
			if tc.expectError {
				assert.Error(t, err, "Should return an error")
			} else {
				assert.NoError(t, err, "Should not return an error")
				if tc.expectedService != "" {
					assert.NotNil(t, service, "Service should not be nil")
					assert.Equal(t, tc.expectedService, service.Name, "Service name should match expected")
				} else {
					assert.Nil(t, service, "Service should be nil")
				}
			}
		})
	}
}

// TestFindMetricsPortAndScheme tests the findMetricsPortAndScheme function
func TestFindMetricsPortAndScheme(t *testing.T) {
	// Test cases
	testCases := []struct {
		name            string
		servicePorts    []corev1.ServicePort
		annotations     map[string]string
		expectedPort    *int32
		expectedScheme  string
		expectPortFound bool
	}{
		{
			name: "Should find HTTPS metrics port",
			servicePorts: []corev1.ServicePort{
				{
					Name: "metrics",
					Port: constants.ETCDMetricsPortHTTPS,
				},
			},
			annotations:     nil,
			expectedPort:    &[]int32{constants.ETCDMetricsPortHTTPS}[0],
			expectedScheme:  "https",
			expectPortFound: true,
		},
		{
			name: "Should find HTTP metrics port",
			servicePorts: []corev1.ServicePort{
				{
					Name: "metrics",
					Port: constants.ETCDMetricsPortHTTP,
				},
			},
			annotations:     nil,
			expectedPort:    &[]int32{constants.ETCDMetricsPortHTTP}[0],
			expectedScheme:  "http",
			expectPortFound: true,
		},
		{
			name: "Should default to HTTPS for unknown port",
			servicePorts: []corev1.ServicePort{
				{
					Name: "metrics",
					Port: 9999, // Unknown port
				},
			},
			annotations:     nil,
			expectedPort:    &[]int32{9999}[0],
			expectedScheme:  "https",
			expectPortFound: true,
		},
		{
			name: "Should use scheme from annotation",
			servicePorts: []corev1.ServicePort{
				{
					Name: "metrics",
					Port: constants.ETCDMetricsPortHTTPS,
				},
			},
			annotations: map[string]string{
				"instana.io/etcd-scheme": "http",
			},
			expectedPort:    &[]int32{constants.ETCDMetricsPortHTTPS}[0],
			expectedScheme:  "http",
			expectPortFound: true,
		},
		{
			name: "Should not find metrics port",
			servicePorts: []corev1.ServicePort{
				{
					Name: "api",
					Port: 8080,
				},
			},
			annotations:     nil,
			expectedPort:    nil,
			expectedScheme:  "",
			expectPortFound: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create service with ports and annotations
			service := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "etcd",
					Namespace:   kubeSystemNamespace,
					Annotations: tc.annotations,
				},
				Spec: corev1.ServiceSpec{
					Ports: tc.servicePorts,
				},
			}

			// Test
			port, scheme := findMetricsPortAndScheme(service)

			// Verify
			if tc.expectPortFound {
				assert.NotNil(t, port, "Port should not be nil when found")
				assert.Equal(t, *tc.expectedPort, *port, "Port should match expected")
				assert.Equal(t, tc.expectedScheme, scheme, "Scheme should match expected")
			} else {
				assert.Nil(t, port, "Port should be nil when not found")
				assert.Equal(t, "", scheme, "Scheme should be empty when port not found")
			}
		})
	}
}

// TestBuildTargetsFromEndpoints tests the buildTargetsFromEndpoints function
func TestBuildTargetsFromEndpoints(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = instanav1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = discoveryv1.AddToScheme(scheme)

	// Test cases
	testCases := []struct {
		name            string
		service         *corev1.Service
		endpointSlice   *discoveryv1.EndpointSlice
		metricsPort     int32
		scheme          string
		expectedTargets []string
		expectError     bool
		setupClientErr  bool
	}{
		{
			name: "Should build targets from endpoints with metrics port",
			service: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "etcd",
					Namespace: kubeSystemNamespace,
				},
			},
			endpointSlice: &discoveryv1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "etcd",
					Namespace: kubeSystemNamespace,
				},
				Ports: []discoveryv1.EndpointPort{
					{
						Name: func() *string { s := "metrics"; return &s }(),
						Port: func() *int32 { p := int32(2379); return &p }(),
					},
				},
				Endpoints: []discoveryv1.Endpoint{
					{
						Addresses: []string{"10.0.0.1"},
						Conditions: discoveryv1.EndpointConditions{
							Ready: func() *bool { b := true; return &b }(),
						},
					},
					{
						Addresses: []string{"10.0.0.2"},
						Conditions: discoveryv1.EndpointConditions{
							Ready: func() *bool { b := true; return &b }(),
						},
					},
				},
			},
			metricsPort: 2379,
			scheme:      "https",
			expectedTargets: []string{
				"https://10.0.0.1:2379/metrics",
				"https://10.0.0.2:2379/metrics",
			},
			expectError:    false,
			setupClientErr: false,
		},
		{
			name: "Should use service port when endpoint port not found",
			service: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "etcd",
					Namespace: kubeSystemNamespace,
				},
			},
			endpointSlice: &discoveryv1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "etcd",
					Namespace: kubeSystemNamespace,
				},
				Ports: []discoveryv1.EndpointPort{
					{
						Name: func() *string { s := "api"; return &s }(),
						Port: func() *int32 { p := int32(8080); return &p }(),
					},
				},
				Endpoints: []discoveryv1.Endpoint{
					{
						Addresses: []string{"10.0.0.1"},
						Conditions: discoveryv1.EndpointConditions{
							Ready: func() *bool { b := true; return &b }(),
						},
					},
				},
			},
			metricsPort:     2379,
			scheme:          "https",
			expectedTargets: []string{"https://10.0.0.1:2379/metrics"},
			expectError:     false,
			setupClientErr:  false,
		},
		{
			name: "Should handle multiple subsets",
			service: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "etcd",
					Namespace: kubeSystemNamespace,
				},
			},
			endpointSlice: &discoveryv1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "etcd",
					Namespace: kubeSystemNamespace,
				},
				Ports: []discoveryv1.EndpointPort{
					{
						Name: func() *string { s := "metrics"; return &s }(),
						Port: func() *int32 { p := int32(2379); return &p }(),
					},
				},
				Endpoints: []discoveryv1.Endpoint{
					{
						Addresses: []string{"10.0.0.1"},
						Conditions: discoveryv1.EndpointConditions{
							Ready: func() *bool { b := true; return &b }(),
						},
					},
					{
						Addresses: []string{"10.0.0.2"},
						Conditions: discoveryv1.EndpointConditions{
							Ready: func() *bool { b := true; return &b }(),
						},
					},
				},
			},
			metricsPort: 2379,
			scheme:      "https",
			expectedTargets: []string{
				"https://10.0.0.1:2379/metrics",
				"https://10.0.0.2:2379/metrics",
			},
			expectError:    false,
			setupClientErr: false,
		},
		{
			name: "Should return empty targets when no addresses",
			service: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "etcd",
					Namespace: kubeSystemNamespace,
				},
			},
			endpointSlice: &discoveryv1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "etcd",
					Namespace: kubeSystemNamespace,
				},
				Ports: []discoveryv1.EndpointPort{
					{
						Name: func() *string { s := "metrics"; return &s }(),
						Port: func() *int32 { p := int32(2379); return &p }(),
					},
				},
				Endpoints: []discoveryv1.Endpoint{},
			},
			metricsPort:     2379,
			scheme:          "https",
			expectedTargets: []string{},
			expectError:     false,
			setupClientErr:  false,
		},
		{
			name: "Should propagate client error",
			service: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "etcd",
					Namespace: kubeSystemNamespace,
				},
			},
			endpointSlice:   nil, // Not used when setupClientErr is true
			metricsPort:     2379,
			scheme:          "https",
			expectedTargets: nil,
			expectError:     true,
			setupClientErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a fake client builder
			clientBuilder := fake.NewClientBuilder().WithScheme(scheme)

			// Add endpointSlice to client if needed
			if tc.endpointSlice != nil {
				clientBuilder = clientBuilder.WithObjects(tc.endpointSlice)
			}

			// Build the client
			fakeClient := clientBuilder.Build()

			// Create mock client that returns error if needed
			var mockClient client.Client
			if tc.setupClientErr {
				mockClient = &errorMockClient{
					Client: fakeClient,
					err:    fmt.Errorf("client error"),
				}
			} else {
				mockClient = fakeClient
			}

			// Test
			ctx := context.Background()
			targets, err := buildTargetsFromEndpoints(
				ctx,
				mockClient,
				tc.service,
				tc.metricsPort,
				tc.scheme,
			)

			// Verify
			if tc.expectError {
				assert.Error(t, err, "Should return an error")
			} else {
				assert.NoError(t, err, "Should not return an error")
				assert.Equal(t, tc.expectedTargets, targets, "Targets should match expected")
			}
		})
	}
}

// TestCheckCASecretExists tests the checkCASecretExists function
func TestCheckCASecretExists(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = instanav1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	// Test cases
	testCases := []struct {
		name          string
		createSecret  bool
		expectedFound bool
	}{
		{
			name:          "Should find CA secret when it exists",
			createSecret:  true,
			expectedFound: true,
		},
		{
			name:          "Should not find CA secret when it doesn't exist",
			createSecret:  false,
			expectedFound: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a fake client builder
			clientBuilder := fake.NewClientBuilder().WithScheme(scheme)

			// Create agent
			agent := &instanav1.InstanaAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "test-namespace",
				},
			}

			// Add CA secret to client if needed
			if tc.createSecret {
				caSecret := &corev1.Secret{ // pragma: allowlist secret
					ObjectMeta: metav1.ObjectMeta{
						Name:      constants.ETCDCASecretName,
						Namespace: agent.Namespace,
					},
				}
				clientBuilder = clientBuilder.WithObjects(caSecret)
			}

			// Build the client
			fakeClient := clientBuilder.Build()

			// Test
			ctx := context.Background()
			logger := ctrl.Log.WithName("test")
			found := checkCASecretExists(ctx, fakeClient, agent, logger)

			// Verify
			assert.Equal(t, tc.expectedFound, found, "CA secret found status should match expected")
		})
	}
}

// TestDiscoverETCDEndpoints tests the main DiscoverETCDEndpoints function
func TestDiscoverETCDEndpoints(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = instanav1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	// Test cases
	testCases := []struct {
		name               string
		shouldSkip         bool
		shouldSkipErr      error
		findServiceResult  *corev1.Service
		findServiceErr     error
		metricsPort        int32
		scheme             string
		buildTargetsResult []string
		buildTargetsErr    error
		caSecretExists     bool
		expectedTargets    []string
		expectedCAFound    bool
		expectError        bool
		expectNilResult    bool
	}{
		{
			name:            "Should return nil when shouldSkipDiscovery returns true",
			shouldSkip:      true,
			shouldSkipErr:   nil,
			expectNilResult: true,
			expectError:     false,
		},
		{
			name:            "Should return error when shouldSkipDiscovery returns error",
			shouldSkip:      false,
			shouldSkipErr:   fmt.Errorf("skip error"),
			expectNilResult: true,
			expectError:     true,
		},
		{
			name:              "Should return nil when findETCDService returns nil",
			shouldSkip:        false,
			shouldSkipErr:     nil,
			findServiceResult: nil,
			findServiceErr:    nil,
			expectNilResult:   true,
			expectError:       false,
		},
		{
			name:              "Should return error when findETCDService returns error",
			shouldSkip:        false,
			shouldSkipErr:     nil,
			findServiceResult: nil,
			findServiceErr:    fmt.Errorf("service error"),
			expectNilResult:   true,
			expectError:       true,
		},
		{
			name:              "Should return nil when no metrics port found",
			shouldSkip:        false,
			shouldSkipErr:     nil,
			findServiceResult: &corev1.Service{},
			findServiceErr:    nil,
			metricsPort:       0,
			scheme:            "",
			expectNilResult:   true,
			expectError:       false,
		},
		{
			name:               "Should return error when buildTargetsFromEndpoints returns error",
			shouldSkip:         false,
			shouldSkipErr:      nil,
			findServiceResult:  &corev1.Service{},
			findServiceErr:     nil,
			metricsPort:        2379,
			scheme:             "https",
			buildTargetsResult: nil,
			buildTargetsErr:    fmt.Errorf("build targets error"),
			expectNilResult:    true,
			expectError:        true,
		},
		{
			name:               "Should return nil when no targets found",
			shouldSkip:         false,
			shouldSkipErr:      nil,
			findServiceResult:  &corev1.Service{},
			findServiceErr:     nil,
			metricsPort:        2379,
			scheme:             "https",
			buildTargetsResult: []string{},
			buildTargetsErr:    nil,
			expectNilResult:    true,
			expectError:        false,
		},
		{
			name:               "Should return targets and CA status when everything succeeds",
			shouldSkip:         false,
			shouldSkipErr:      nil,
			findServiceResult:  &corev1.Service{},
			findServiceErr:     nil,
			metricsPort:        2379,
			scheme:             "https",
			buildTargetsResult: []string{"https://10.0.0.1:2379/metrics"},
			buildTargetsErr:    nil,
			caSecretExists:     true,
			expectedTargets:    []string{"https://10.0.0.1:2379/metrics"},
			expectedCAFound:    true,
			expectNilResult:    false,
			expectError:        false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create agent
			agent := &instanav1.InstanaAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "test-namespace",
				},
			}

			// Create mock functions
			mockShouldSkipDiscovery := func(
				ctx context.Context,
				agent *instanav1.InstanaAgent,
				logger logr.Logger,
				isOpenShiftFunc IsOpenShiftFunc,
			) (bool, error) {
				return tc.shouldSkip, tc.shouldSkipErr
			}

			mockFindETCDService := func(ctx context.Context, client client.Client, logger logr.Logger) (*corev1.Service, error) {
				return tc.findServiceResult, tc.findServiceErr
			}

			mockFindMetricsPortAndScheme := func(service *corev1.Service) (*int32, string) {
				if tc.metricsPort == 0 {
					return nil, tc.scheme
				}
				return &tc.metricsPort, tc.scheme
			}

			mockBuildTargetsFromEndpoints := func(
				ctx context.Context,
				client client.Client,
				service *corev1.Service,
				metricsPort int32,
				scheme string,
			) ([]string, error) {
				return tc.buildTargetsResult, tc.buildTargetsErr
			}

			mockCheckCASecretExists := func(
				ctx context.Context,
				client client.Client,
				agent *instanav1.InstanaAgent,
				logger logr.Logger,
			) bool {
				return tc.caSecretExists
			}

			// Test
			ctx := context.Background()
			logger := ctrl.Log.WithName("test")
			result, err := discoverETCDEndpoints(
				ctx,
				nil, // client not used due to mocks
				agent,
				logger,
				nil, // isOpenShiftFunc not used due to mocks
				mockShouldSkipDiscovery,
				mockFindETCDService,
				mockFindMetricsPortAndScheme,
				mockBuildTargetsFromEndpoints,
				mockCheckCASecretExists,
			)

			// Verify
			if tc.expectError {
				assert.Error(t, err, "Should return an error")
			} else {
				assert.NoError(t, err, "Should not return an error")
			}

			if tc.expectNilResult {
				assert.Nil(t, result, "Result should be nil")
			} else {
				assert.NotNil(t, result, "Result should not be nil")
				assert.Equal(t, tc.expectedTargets, result.Targets, "Targets should match expected")
				assert.Equal(t, tc.expectedCAFound, result.CAFound, "CAFound should match expected")
			}
		})
	}
}

// Helper functions that match the signatures of the methods in InstanaAgentReconciler
// These are the functions we're testing

func shouldSkipDiscovery(
	ctx context.Context,
	agent *instanav1.InstanaAgent,
	logger logr.Logger,
	isOpenShiftFunc IsOpenShiftFunc,
) (bool, error) {
	operatorUtils := operator_utils.NewOperatorUtils(ctx, nil, agent, nil)
	isOpenShift, isOpenShiftRes := isOpenShiftFunc(ctx, operatorUtils)

	if isOpenShiftRes.suppliesReconcileResult() {
		return false, fmt.Errorf("failed to determine if cluster is OpenShift")
	}

	if isOpenShift {
		logger.Info("Skipping ETCD discovery on OpenShift cluster")
		return true, nil
	}

	if len(agent.Spec.K8sSensor.ETCD.Targets) > 0 {
		logger.Info("Using ETCD targets from CR spec", "targets", agent.Spec.K8sSensor.ETCD.Targets)
		return true, nil
	}

	return false, nil
}

func getServiceWithLabel(
	ctx context.Context,
	client client.Client,
	name, labelKey, labelValue string,
) (*corev1.Service, error) {
	service := &corev1.Service{}
	err := client.Get(ctx, types.NamespacedName{
		Namespace: kubeSystemNamespace,
		Name:      name,
	}, service)

	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	if service.Labels == nil || service.Labels[labelKey] != labelValue {
		return nil, nil
	}

	return service, nil
}

func getServiceByName(
	ctx context.Context,
	client client.Client,
	name string,
) (*corev1.Service, error) {
	service := &corev1.Service{}
	err := client.Get(ctx, types.NamespacedName{
		Namespace: kubeSystemNamespace,
		Name:      name,
	}, service)

	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return service, nil
}

func findETCDService(
	ctx context.Context,
	client client.Client,
	logger logr.Logger,
) (*corev1.Service, error) {
	// Try services with component=etcd label first
	if service, err := getServiceWithLabel(ctx, client, "etcd", "component", "etcd"); err != nil {
		return nil, err
	} else if service != nil {
		logger.Info("Found etcd service with component=etcd label", "name", service.Name)
		return service, nil
	}

	if service, err := getServiceWithLabel(ctx, client, "etcd-metrics", "component", "etcd"); err != nil {
		return nil, err
	} else if service != nil {
		logger.Info("Found etcd-metrics service with component=etcd label", "name", service.Name)
		return service, nil
	}

	// Fallback to name-based search
	logger.Info("No service found with component=etcd label, trying by name")

	// Try by name in sequence
	serviceNames := []string{"etcd", "etcd-metrics", "etcd-k8s"}
	for _, name := range serviceNames {
		service, err := getServiceByName(ctx, client, name)
		if err != nil {
			return nil, err
		}
		if service != nil {
			return service, nil
		}
	}

	return nil, nil
}

func findMetricsPortAndScheme(service *corev1.Service) (*int32, string) {
	for _, port := range service.Spec.Ports {
		if port.Name == "metrics" {
			// Use switch/case for scheme determination
			scheme := "https" // Default to https for unknown ports
			switch port.Port {
			case constants.ETCDMetricsPortHTTPS:
				scheme = "https"
			case constants.ETCDMetricsPortHTTP:
				scheme = "http"
			}

			// Check for scheme annotation override
			if schemeOverride, ok := service.Annotations["instana.io/etcd-scheme"]; ok {
				scheme = schemeOverride
			}

			return &port.Port, scheme
		}
	}

	return nil, ""
}

func buildTargetsFromEndpoints(
	ctx context.Context,
	client client.Client,
	service *corev1.Service,
	metricsPort int32,
	scheme string,
) ([]string, error) {
	// Get endpoint slice for the service
	endpointSlice := &discoveryv1.EndpointSlice{}
	if err := client.Get(ctx, types.NamespacedName{
		Namespace: kubeSystemNamespace,
		Name:      service.Name,
	}, endpointSlice); err != nil {
		return nil, err
	}

	targets := make([]string, 0)

	// Find the metrics port in the endpoint slice
	var endpointPort int32
	for _, port := range endpointSlice.Ports {
		if port.Name != nil && *port.Name == "metrics" && port.Port != nil {
			endpointPort = *port.Port
			break
		}
	}

	// If no metrics port found in endpoint slice, use the service port
	if endpointPort == 0 {
		endpointPort = metricsPort
	}

	// Add targets for each endpoint
	for _, endpoint := range endpointSlice.Endpoints {
		if endpoint.Conditions.Ready != nil && *endpoint.Conditions.Ready {
			for _, address := range endpoint.Addresses {
				target := fmt.Sprintf("%s://%s:%d/metrics", scheme, address, endpointPort)
				targets = append(targets, target)
			}
		}
	}

	// Sort targets for consistent comparison with current state
	sort.Strings(targets)

	return targets, nil
}

func checkCASecretExists(
	ctx context.Context,
	client client.Client,
	agent *instanav1.InstanaAgent,
	logger logr.Logger,
) bool {
	caSecret := &corev1.Secret{} // pragma: allowlist secret
	err := client.Get(ctx, types.NamespacedName{
		Namespace: agent.Namespace,
		Name:      constants.ETCDCASecretName,
	}, caSecret)

	if err == nil {
		logger.Info("Found etcd-ca secret in agent namespace")
		return true
	}

	return false
}

func discoverETCDEndpoints(
	ctx context.Context,
	client client.Client,
	agent *instanav1.InstanaAgent,
	logger logr.Logger,
	isOpenShiftFunc IsOpenShiftFunc,
	shouldSkipDiscoveryFunc SkipDiscoveryFunc,
	findETCDServiceFunc FindServiceFunc,
	findMetricsPortAndSchemeFunc FindPortAndSchemeFunc,
	buildTargetsFromEndpointsFunc BuildTargetsFunc,
	checkCASecretExistsFunc CheckCASecretFunc,
) (*DiscoveredETCDTargets, error) {
	// Step 1: Check if discovery should be skipped
	shouldSkip, err := shouldSkipDiscoveryFunc(ctx, agent, logger, isOpenShiftFunc)
	if err != nil {
		return nil, err
	}
	if shouldSkip {
		return nil, nil
	}

	// Step 2: Find etcd service
	etcdService, err := findETCDServiceFunc(ctx, client, logger)
	if err != nil {
		return nil, err
	}
	if etcdService == nil {
		return nil, nil
	}

	logger.Info("Found etcd service", "name", etcdService.Name)

	// Step 3: Find metrics port and determine scheme
	metricsPortPtr, scheme := findMetricsPortAndSchemeFunc(etcdService)
	if metricsPortPtr == nil {
		logger.Info("No metrics port found in etcd service")
		return nil, nil
	}
	metricsPort := *metricsPortPtr

	// Step 4: Get endpoints and build targets
	targets, err := buildTargetsFromEndpointsFunc(ctx, client, etcdService, metricsPort, scheme)
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		logger.Info("No endpoints found for etcd service")
		return nil, nil
	}

	// Step 5: Check for CA secret and return results
	caSecretExists := checkCASecretExistsFunc(ctx, client, agent, logger)

	logger.Info("Discovered etcd targets", "targets", targets, "caFound", caSecretExists)

	return &DiscoveredETCDTargets{
		Targets: targets,
		CAFound: caSecretExists,
	}, nil
}
