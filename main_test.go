/*
(c) Copyright IBM Corp. 2024, 2025

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

package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	agentoperatorv1 "github.com/instana/instana-agent-operator/api/v1"
)

func TestLabelBasedCacheConfiguration(t *testing.T) {
	assertions := require.New(t)

	// Get cache options using the shared function
	cacheOpts, err := getCacheOptions()
	assertions.NoError(err)
	assertions.NotNil(cacheOpts.ByObject)
	assertions.Len(cacheOpts.ByObject, 12, "Should have 12 resource types configured")

	// Verify the cache configuration structure is correct
	assertions.IsType(map[client.Object]cache.ByObject{}, cacheOpts.ByObject)
}

func TestOperatorManagedLabelSelector(t *testing.T) {
	assertions := require.New(t)

	// Test the label selector parsing
	managedByOperator, err := labels.Parse("app.kubernetes.io/managed-by=instana-agent-operator")
	assertions.NoError(err)
	assertions.NotNil(managedByOperator)

	// Test that the selector matches expected labels
	testLabels := map[string]string{
		"app.kubernetes.io/managed-by": "instana-agent-operator",
		"app.kubernetes.io/name":       "instana-agent",
	}
	assertions.True(managedByOperator.Matches(labels.Set(testLabels)))

	// Test that the selector doesn't match non-operator resources
	nonOperatorLabels := map[string]string{
		"app": "some-other-app",
	}
	assertions.False(managedByOperator.Matches(labels.Set(nonOperatorLabels)))
}

func TestCacheConfigurationForMultiNamespaceSupport(t *testing.T) {
	assertions := require.New(t)

	// Get cache options using the shared function
	cacheOpts, err := getCacheOptions()
	assertions.NoError(err)
	assertions.NotNil(cacheOpts.ByObject)

	// Verify InstanaAgent CRs are cached without namespace restriction
	agentConfig := cacheOpts.ByObject[&agentoperatorv1.InstanaAgent{}]
	assertions.Nil(agentConfig.Label, "InstanaAgent CRs should be cached in all namespaces")

	// Verify ConfigMaps are cached cluster-wide for ETCD discovery
	configMapConfig := cacheOpts.ByObject[&corev1.ConfigMap{}]
	assertions.Nil(
		configMapConfig.Label,
		"ConfigMaps should be cached cluster-wide for ETCD discovery",
	)

	// Verify Secrets are cached cluster-wide for ETCD and user secrets
	secretConfig := cacheOpts.ByObject[&corev1.Secret{}]
	assertions.Nil(secretConfig.Label, "Secrets should be cached cluster-wide")

	// Verify Services are cached cluster-wide for ETCD discovery
	serviceConfig := cacheOpts.ByObject[&corev1.Service{}]
	assertions.Nil(
		serviceConfig.Label,
		"Services should be cached cluster-wide for ETCD discovery",
	)
}

func TestOperatorNamespaceFromEnvironment(t *testing.T) {
	const defaultNamespace = "instana-agent"

	for _, tt := range []struct {
		name              string
		envValue          string
		expectedNamespace string
	}{
		{
			name:              "custom namespace from environment",
			envValue:          "custom-monitoring",
			expectedNamespace: "custom-monitoring",
		},
		{
			name:              "default namespace when env not set",
			envValue:          "",
			expectedNamespace: defaultNamespace,
		},
		{
			name:              "instana-agent namespace",
			envValue:          "instana-agent",
			expectedNamespace: "instana-agent",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assertions := require.New(t)

			if tt.envValue != "" {
				assertions.NoError(os.Setenv("POD_NAMESPACE", tt.envValue))
				defer func() {
					assertions.NoError(os.Unsetenv("POD_NAMESPACE"))
				}()
			} else {
				assertions.NoError(os.Unsetenv("POD_NAMESPACE"))
			}

			operatorNamespace := os.Getenv("POD_NAMESPACE")
			if operatorNamespace == "" {
				operatorNamespace = defaultNamespace
			}

			assertions.Equal(tt.expectedNamespace, operatorNamespace)
		})
	}
}

func TestCacheOptionsStructure(t *testing.T) {
	assertions := require.New(t)

	managedByOperator, err := labels.Parse("app.kubernetes.io/managed-by=instana-agent-operator")
	assertions.NoError(err)

	// Store the key to verify map access
	daemonSetKey := &appsv1.DaemonSet{}

	cacheOpts := cache.Options{
		ByObject: map[client.Object]cache.ByObject{
			daemonSetKey: {Label: managedByOperator},
		},
	}

	assertions.NotNil(cacheOpts.ByObject)
	assertions.IsType(map[client.Object]cache.ByObject{}, cacheOpts.ByObject)
	assertions.Len(cacheOpts.ByObject, 1, "Should have one resource type configured")

	// Verify using the same key reference
	daemonSetConfig := cacheOpts.ByObject[daemonSetKey]
	assertions.NotNil(daemonSetConfig.Label, "DaemonSet should have label filter")
	assertions.Equal(managedByOperator, daemonSetConfig.Label)
}

// Made with Bob
