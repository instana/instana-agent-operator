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
package v1

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/instana/instana-agent-operator/pkg/pointer"
)

func TestInstanaAgentRemote_Default(t *testing.T) {
	defaultTrue := pointer.To(true)

	tests := []struct {
		name     string
		spec     *InstanaAgentRemoteSpec
		expected *InstanaAgentRemoteSpec
	}{
		{
			name: "agent_setup",
			spec: &InstanaAgentRemoteSpec{
				Zone: Name{
					"test",
				},
				Agent: BaseAgentSpec{
					EndpointHost: "custom-host.instana.io",
					EndpointPort: "8443",
					ExtendedImageSpec: ExtendedImageSpec{
						ImageSpec: ImageSpec{
							Name:       "custom/agent",
							Tag:        "1.2.3",
							PullPolicy: corev1.PullIfNotPresent,
						},
					},
					ConfigurationYaml: "remote-config-yaml",
					Key:               "agent-key-123",
					DownloadKey:       "download-key",
				},
			},
			expected: &InstanaAgentRemoteSpec{
				Zone: Name{
					"test",
				},
				Agent: BaseAgentSpec{
					EndpointHost: "custom-host.instana.io",
					EndpointPort: "8443",
					ExtendedImageSpec: ExtendedImageSpec{
						ImageSpec: ImageSpec{
							Name:       "custom/agent",
							Tag:        "1.2.3",
							PullPolicy: corev1.PullIfNotPresent,
						},
					},
					Key:               "agent-key-123",
					DownloadKey:       "download-key",
					ConfigurationYaml: "remote-config-yaml",
				},
				Rbac: Create{Create: defaultTrue},
				ServiceAccountSpec: ServiceAccountSpec{
					Create: Create{Create: defaultTrue},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertions := require.New(t)

			ra := &InstanaAgentRemote{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "instana-agent-remote",
					Namespace: "default",
				},
				Spec: *tt.spec,
			}
			ra.Default()

			assertions.Equal(tt.expected, &ra.Spec)
		})
	}
}
