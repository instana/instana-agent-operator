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

package volume

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
)

func TestRemoteVolumeBuilder_SecretsVolume(t *testing.T) {
	// Test case 1: useSecretMounts is true (default)
	t.Run("with useSecretMounts enabled", func(t *testing.T) {
		// Setup
		agent := &instanav1.InstanaAgentRemote{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-agent",
			},
			Spec: instanav1.InstanaAgentRemoteSpec{
				Agent: instanav1.BaseAgentSpec{
					KeysSecret:  "test-secret",
					DownloadKey: "test-download-key", // Add download key for this test
				},
			},
		}
		builder := NewVolumeBuilderRemote(agent)

		// Execute
		volumes, mounts := builder.Build(SecretsVolumeRemote)

		// Verify
		require.Len(t, volumes, 1)
		require.Len(t, mounts, 1)

		// Check volume
		volume := volumes[0]
		assert.Equal(t, "instana-secrets", volume.Name)
		assert.Equal(t, "test-secret", volume.Secret.SecretName)
		assert.NotNil(t, volume.Secret.Items)

		// Verify key mappings
		foundAgentKey := false
		foundDownloadKey := false
		for _, item := range volume.Secret.Items {
			if item.Key == constants.AgentKey && item.Path == constants.SecretFileAgentKey {
				foundAgentKey = true
			}
			if item.Key == constants.DownloadKey && item.Path == constants.SecretFileDownloadKey {
				foundDownloadKey = true
			}
		}
		assert.True(t, foundAgentKey, "Agent key mapping not found")
		assert.True(t, foundDownloadKey, "Download key mapping not found")

		// Check mount
		mount := mounts[0]
		assert.Equal(t, "instana-secrets", mount.Name)
		assert.Equal(t, constants.InstanaSecretsDirectory, mount.MountPath)
		assert.True(t, mount.ReadOnly)
	})

	// Test case 2: useSecretMounts is false
	t.Run("with useSecretMounts disabled", func(t *testing.T) {
		// Setup
		useSecretMounts := false
		agent := &instanav1.InstanaAgentRemote{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-agent",
			},
			Spec: instanav1.InstanaAgentRemoteSpec{
				UseSecretMounts: &useSecretMounts,
				Agent: instanav1.BaseAgentSpec{
					KeysSecret: "test-secret",
				},
			},
		}
		builder := NewVolumeBuilderRemote(agent)

		// Execute
		volumes, mounts := builder.Build(SecretsVolumeRemote)

		// Verify
		assert.Empty(t, volumes)
		assert.Empty(t, mounts)
	})

	// Test case 3: with proxy configuration
	t.Run("with proxy configuration", func(t *testing.T) {
		// Setup
		agent := &instanav1.InstanaAgentRemote{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-agent",
			},
			Spec: instanav1.InstanaAgentRemoteSpec{
				Agent: instanav1.BaseAgentSpec{
					KeysSecret:    "test-secret",
					ProxyHost:     "proxy.example.com",
					ProxyUser:     "proxyuser",
					ProxyPassword: "proxypass",
				},
			},
		}
		builder := NewVolumeBuilderRemote(agent)

		// Execute
		volumes, _ := builder.Build(SecretsVolumeRemote)

		// Verify
		require.Len(t, volumes, 1)
		volume := volumes[0]

		// Verify proxy-related key mappings
		foundProxyUser := false
		foundProxyPassword := false
		foundHttpsProxy := false
		for _, item := range volume.Secret.Items {
			if item.Key == constants.SecretKeyProxyUser &&
				item.Path == constants.SecretFileProxyUser {
				foundProxyUser = true
			}
			if item.Key == constants.SecretKeyProxyPassword &&
				item.Path == constants.SecretFileProxyPassword {
				foundProxyPassword = true
			}
			if item.Key == constants.SecretKeyHttpsProxy &&
				item.Path == constants.SecretFileHttpsProxy {
				foundHttpsProxy = true
			}
		}
		assert.True(t, foundProxyUser, "Proxy user mapping not found")
		assert.True(t, foundProxyPassword, "Proxy password mapping not found")
		assert.True(t, foundHttpsProxy, "HTTPS_PROXY mapping not found")
	})

	// Test case 4: with repository mirror credentials
	t.Run("with repository mirror credentials", func(t *testing.T) {
		// Setup
		agent := &instanav1.InstanaAgentRemote{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-agent",
			},
			Spec: instanav1.InstanaAgentRemoteSpec{
				Agent: instanav1.BaseAgentSpec{
					KeysSecret:                "test-secret",
					MirrorReleaseRepoUsername: "releaseuser",
					MirrorReleaseRepoPassword: "releasepass",
					MirrorSharedRepoUsername:  "shareduser",
					MirrorSharedRepoPassword:  "sharedpass",
				},
			},
		}
		builder := NewVolumeBuilderRemote(agent)

		// Execute
		volumes, _ := builder.Build(SecretsVolumeRemote)

		// Verify
		require.Len(t, volumes, 1)
		volume := volumes[0]

		// Verify mirror-related key mappings
		foundReleaseRepoUsername := false
		foundReleaseRepoPassword := false
		foundSharedRepoUsername := false
		foundSharedRepoPassword := false
		for _, item := range volume.Secret.Items {
			if item.Key == constants.SecretKeyMirrorReleaseRepoUsername &&
				item.Path == constants.SecretFileMirrorReleaseRepoUsername {
				foundReleaseRepoUsername = true
			}
			if item.Key == constants.SecretKeyMirrorReleaseRepoPassword &&
				item.Path == constants.SecretFileMirrorReleaseRepoPassword {
				foundReleaseRepoPassword = true
			}
			if item.Key == constants.SecretKeyMirrorSharedRepoUsername &&
				item.Path == constants.SecretFileMirrorSharedRepoUsername {
				foundSharedRepoUsername = true
			}
			if item.Key == constants.SecretKeyMirrorSharedRepoPassword &&
				item.Path == constants.SecretFileMirrorSharedRepoPassword {
				foundSharedRepoPassword = true
			}
		}
		assert.True(t, foundReleaseRepoUsername, "Mirror release repo username mapping not found")
		assert.True(t, foundReleaseRepoPassword, "Mirror release repo password mapping not found")
		assert.True(t, foundSharedRepoUsername, "Mirror shared repo username mapping not found")
		assert.True(t, foundSharedRepoPassword, "Mirror shared repo password mapping not found")
	})
}

// Made with Bob
