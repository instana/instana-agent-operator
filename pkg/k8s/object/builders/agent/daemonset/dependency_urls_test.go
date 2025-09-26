/*
(c) Copyright IBM Corp. 2024, 2025
*/

package daemonset

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/status"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

func TestDaemonSetBuilder_InitContainers_WithDependencyURLs(t *testing.T) {
	// Create agent with dependencyURLs
	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
		},
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				Key: "test-key",
				DependencyURLs: []string{
					"https://example.com/test-file1.jar",
					"https://example.com/test-file2.jar",
				},
			},
			Cluster: instanav1.Name{
				Name: "test-cluster",
			},
		},
	}

	// Create builder
	builder := NewDaemonSetBuilder(agent, false, status.NewAgentStatusManager())

	// Build the DaemonSet
	result := builder.Build()
	require.True(t, result.IsPresent())

	// Get the DaemonSet
	ds := result.Get().(*appsv1.DaemonSet).Spec.Template.Spec

	// Verify init containers
	assert.Equal(t, 1, len(ds.InitContainers))
	assert.Equal(t, "init-dependency-downloader", ds.InitContainers[0].Name)
	assert.Contains(t, ds.InitContainers[0].Command[2], "https://example.com/test-file1.jar")
	assert.Contains(t, ds.InitContainers[0].Command[2], "https://example.com/test-file2.jar")

	// Verify volume mounts
	foundVolume := false
	for _, volume := range ds.Volumes {
		if volume.Name == "instanadeploy" {
			foundVolume = true
			assert.NotNil(t, volume.EmptyDir)
			break
		}
	}
	assert.True(t, foundVolume, "instanadeploy volume not found")

	// Verify agent container has the volume mount
	foundVolumeMount := false
	for _, volumeMount := range ds.Containers[0].VolumeMounts {
		if volumeMount.Name == "instanadeploy" {
			foundVolumeMount = true
			assert.Equal(t, "/opt/instana/agent/deploy", volumeMount.MountPath)
			break
		}
	}
	assert.True(t, foundVolumeMount, "instanadeploy volume mount not found in agent container")
}

func TestDaemonSetBuilder_InitContainers_WithoutDependencyURLs(t *testing.T) {
	// Create agent without dependencyURLs
	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
		},
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				Key: "test-key",
			},
			Cluster: instanav1.Name{
				Name: "test-cluster",
			},
		},
	}

	// Create builder
	builder := NewDaemonSetBuilder(agent, false, status.NewAgentStatusManager())

	// Build the DaemonSet
	result := builder.Build()
	require.True(t, result.IsPresent())

	// Get the DaemonSet
	ds := result.Get().(*appsv1.DaemonSet).Spec.Template.Spec

	// Verify no init containers
	assert.Equal(t, 0, len(ds.InitContainers))

	// Verify no instanadeploy volume
	foundVolume := false
	for _, volume := range ds.Volumes {
		if volume.Name == "instanadeploy" {
			foundVolume = true
			break
		}
	}
	assert.False(t, foundVolume, "instanadeploy volume should not be present")
}

// Made with Bob
