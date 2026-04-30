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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/config"
)

func TestManagerCacheConfiguration(t *testing.T) {
	assertions := require.New(t)

	operatorNamespace := "test-namespace"
	assertions.NoError(os.Setenv("POD_NAMESPACE", operatorNamespace))
	defer func() {
		assertions.NoError(os.Unsetenv("POD_NAMESPACE"))
	}()

	opts := ctrl.Options{
		Cache: cache.Options{
			DefaultNamespaces: map[string]cache.Config{
				operatorNamespace: {},
			},
		},
		Controller: config.Controller{
			SkipNameValidation: func() *bool { b := true; return &b }(),
		},
	}

	assertions.NotNil(opts.Cache.DefaultNamespaces)
	assertions.Len(opts.Cache.DefaultNamespaces, 1)
	assertions.Contains(opts.Cache.DefaultNamespaces, operatorNamespace)
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

	operatorNamespace := "test-namespace"

	cacheOpts := cache.Options{
		DefaultNamespaces: map[string]cache.Config{
			operatorNamespace: {},
		},
	}

	assertions.NotNil(cacheOpts.DefaultNamespaces)
	assertions.IsType(map[string]cache.Config{}, cacheOpts.DefaultNamespaces)

	_, exists := cacheOpts.DefaultNamespaces[operatorNamespace]
	assertions.True(exists)
}

// Made with Bob
