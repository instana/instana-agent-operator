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
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	instanaclient "github.com/instana/instana-agent-operator/pkg/k8s/client"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/operator_utils"
)

// Mock client that returns errors for specific operations
type errorMockClient struct {
	client.Client
	err error
}

// Get implements client.Client
func (c *errorMockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object,
	opts ...client.GetOption,
) error {
	return c.err
}

// List implements client.Client
func (c *errorMockClient) List(
	ctx context.Context,
	list client.ObjectList,
	opts ...client.ListOption,
) error {
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
	opts ...client.GetOption,
) error {
	if key.Name == c.errorForName {
		return c.err
	}
	return c.Client.Get(ctx, key, obj, opts...)
}

// TestShouldSkipDiscovery tests the ShouldSkipDiscovery function
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

			// Create real reconciler
			reconciler := &InstanaAgentReconciler{
				client: instanaclient.NewInstanaAgentClient(fakeClient),
				scheme: scheme,
			}

			// Mock isOpenShift function
			mockIsOpenShift := func(
				r *InstanaAgentReconciler,
				ctx context.Context,
				operatorUtils operator_utils.OperatorUtils,
			) (bool, reconcileReturn) {
				if tc.isOpenShiftErr {
					return false, reconcileFailure(fmt.Errorf("isOpenShift error"))
				}
				return tc.isOpenShift, reconcileContinue()
			}
			originalIsOpenShift := IsOpenShift
			defer func() {
				IsOpenShift = originalIsOpenShift
			}()
			IsOpenShift = mockIsOpenShift

			// Test
			ctx := context.Background()
			skip, err := ShouldSkipDiscovery(reconciler, ctx, agent)

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

// TestGetServiceWithLabel tests the GetServiceWithLabel function
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

			// Create real reconciler
			reconciler := &InstanaAgentReconciler{
				client: instanaclient.NewInstanaAgentClient(mockClient),
				scheme: scheme,
			}

			// Test
			ctx := context.Background()
			service, err := GetServiceWithLabel(
				reconciler,
				ctx,
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

// TestGetServiceByName tests the GetServiceByName function
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

			// Create real reconciler
			reconciler := &InstanaAgentReconciler{
				client: instanaclient.NewInstanaAgentClient(mockClient),
				scheme: scheme,
			}

			// Test
			ctx := context.Background()
			service, err := GetServiceByName(reconciler, ctx, tc.serviceName)

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
			name: "Should propagate error from GetServiceWithLabel",
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

			// Create real reconciler
			reconciler := &InstanaAgentReconciler{
				client: instanaclient.NewInstanaAgentClient(mockClient),
				scheme: scheme,
			}

			// Test
			ctx := context.Background()
			logger := ctrl.Log.WithName("test")
			service, err := FindETCDService(reconciler, ctx, logger)

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

			// Create real reconciler
			clientScheme := runtime.NewScheme()
			fakeClient := fake.NewClientBuilder().WithScheme(clientScheme).Build()
			reconciler := &InstanaAgentReconciler{
				client: instanaclient.NewInstanaAgentClient(fakeClient),
				scheme: clientScheme,
			}

			// Test
			port, scheme := FindMetricsPortAndScheme(reconciler, service)

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
		legacyEndpoints *corev1.Endpoints
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
					Labels: map[string]string{
						discoveryv1.LabelServiceName: "etcd",
					},
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
					Labels: map[string]string{
						discoveryv1.LabelServiceName: "etcd",
					},
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
					Labels: map[string]string{
						discoveryv1.LabelServiceName: "etcd",
					},
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
					Labels: map[string]string{
						discoveryv1.LabelServiceName: "etcd",
					},
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
			name: "Should fall back to legacy Endpoints when no EndpointSlices",
			service: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "etcd",
					Namespace: kubeSystemNamespace,
				},
			},
			legacyEndpoints: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "etcd",
					Namespace: kubeSystemNamespace,
				},
				Subsets: []corev1.EndpointSubset{
					{
						Addresses: []corev1.EndpointAddress{
							{IP: "10.0.0.3"},
						},
						Ports: []corev1.EndpointPort{
							{
								Name: "metrics",
								Port: 2379,
							},
						},
					},
				},
			},
			metricsPort:     2379,
			scheme:          "https",
			expectedTargets: []string{"https://10.0.0.3:2379/metrics"},
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

			if tc.legacyEndpoints != nil {
				clientBuilder = clientBuilder.WithObjects(tc.legacyEndpoints)
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

			// Create real reconciler
			reconciler := &InstanaAgentReconciler{
				client: instanaclient.NewInstanaAgentClient(mockClient),
				scheme: scheme,
			}

			// Test
			ctx := context.Background()
			targets, err := BuildTargetsFromEndpoints(
				reconciler,
				ctx,
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

// TestCheckCASecretExists tests the CheckCASecretExists function
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

			// Create real reconciler
			reconciler := &InstanaAgentReconciler{
				client: instanaclient.NewInstanaAgentClient(fakeClient),
				scheme: scheme,
			}

			// Test
			ctx := context.Background()
			found := CheckCASecretExists(reconciler, ctx, agent)

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
				r *InstanaAgentReconciler,
				ctx context.Context,
				agent *instanav1.InstanaAgent,
			) (bool, error) {
				return tc.shouldSkip, tc.shouldSkipErr
			}
			originalShouldSkipDiscovery := ShouldSkipDiscovery
			defer func() {
				ShouldSkipDiscovery = originalShouldSkipDiscovery
			}()
			ShouldSkipDiscovery = mockShouldSkipDiscovery

			mockFindETCDService := func(
				r *InstanaAgentReconciler, ctx context.Context, logger logr.Logger,
			) (*corev1.Service, error) {
				return tc.findServiceResult, tc.findServiceErr
			}
			originalFindETCDService := FindETCDService
			defer func() {
				FindETCDService = originalFindETCDService
			}()
			FindETCDService = mockFindETCDService

			mockFindMetricsPortAndScheme := func(r *InstanaAgentReconciler, service *corev1.Service) (*int32, string) {
				if tc.metricsPort == 0 {
					return nil, tc.scheme
				}
				return &tc.metricsPort, tc.scheme
			}
			originalFindMetricsPortAndScheme := FindMetricsPortAndScheme
			defer func() {
				FindMetricsPortAndScheme = originalFindMetricsPortAndScheme
			}()
			FindMetricsPortAndScheme = mockFindMetricsPortAndScheme

			mockBuildTargetsFromEndpoints := func(
				r *InstanaAgentReconciler,
				ctx context.Context,
				service *corev1.Service,
				metricsPort int32,
				scheme string,
			) ([]string, error) {
				return tc.buildTargetsResult, tc.buildTargetsErr
			}
			originalBuildTargetsFromEndpoints := BuildTargetsFromEndpoints
			defer func() {
				BuildTargetsFromEndpoints = originalBuildTargetsFromEndpoints
			}()
			BuildTargetsFromEndpoints = mockBuildTargetsFromEndpoints

			mockCheckCASecretExists := func(
				r *InstanaAgentReconciler,
				ctx context.Context,
				agent *instanav1.InstanaAgent,
			) bool {
				return tc.caSecretExists
			}
			originalCheckCASecretExists := CheckCASecretExists
			defer func() {
				CheckCASecretExists = originalCheckCASecretExists
			}()
			CheckCASecretExists = mockCheckCASecretExists

			// Create real reconciler
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
			reconciler := &InstanaAgentReconciler{
				client: instanaclient.NewInstanaAgentClient(fakeClient),
				scheme: scheme,
			}

			// Test
			ctx := context.Background()
			result, err := reconciler.DiscoverETCDEndpoints(
				ctx,
				agent,
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
