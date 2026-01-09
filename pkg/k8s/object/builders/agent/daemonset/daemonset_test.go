/*
(c) Copyright IBM Corp. 2025, 2026

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

package daemonset

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/internal/mocks"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/ports"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/status"
)

func TestGetLivenessProbe_DefaultValues(t *testing.T) {
	// Create a minimal InstanaAgent without custom liveness probe
	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
		},
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				EndpointHost: "ingress-red-saas.instana.io",
				EndpointPort: "443",
				Key:          "test-key",
			},
			Cluster: instanav1.Name{
				Name: "test-cluster",
			},
		},
	}

	// Apply defaults
	agent.Default()

	// Create the builder
	mockClient := &mocks.MockInstanaAgentClient{}
	eventRecorder := record.NewFakeRecorder(10)
	statusManager := status.NewAgentStatusManager(mockClient, eventRecorder)
	builder := NewDaemonSetBuilder(agent, false, statusManager).(*daemonSetBuilder)

	// Get the liveness probe
	probe := builder.getLivenessProbe()

	// Assert default values
	require.NotNil(t, probe)
	assert.Equal(t, int32(600), probe.InitialDelaySeconds)
	assert.Equal(t, int32(5), probe.TimeoutSeconds)
	assert.Equal(t, int32(10), probe.PeriodSeconds)
	assert.Equal(t, int32(3), probe.FailureThreshold)

	// Assert HTTPGet configuration
	require.NotNil(t, probe.HTTPGet)
	assert.Equal(t, "127.0.0.1", probe.HTTPGet.Host)
	assert.Equal(t, "/status", probe.HTTPGet.Path)
	assert.Equal(t, intstr.FromInt32(ports.InstanaAgentAPIPortConfig.Port), probe.HTTPGet.Port)
}

func TestGetLivenessProbe_CustomValues(t *testing.T) {
	// Create a custom liveness probe
	customProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Host: "127.0.0.1",
				Path: "/health",
				Port: intstr.FromInt32(42699),
			},
		},
		InitialDelaySeconds: 900,
		TimeoutSeconds:      10,
		PeriodSeconds:       15,
		FailureThreshold:    5,
	}

	// Create an InstanaAgent with custom liveness probe
	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
		},
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				EndpointHost: "ingress-red-saas.instana.io",
				EndpointPort: "443",
				Key:          "test-key",
				Pod: instanav1.AgentPodSpec{
					LivenessProbe: customProbe,
				},
			},
			Cluster: instanav1.Name{
				Name: "test-cluster",
			},
		},
	}

	// Apply defaults
	agent.Default()

	// Create the builder
	mockClient := &mocks.MockInstanaAgentClient{}
	eventRecorder := record.NewFakeRecorder(10)
	statusManager := status.NewAgentStatusManager(mockClient, eventRecorder)
	builder := NewDaemonSetBuilder(agent, false, statusManager).(*daemonSetBuilder)

	// Get the liveness probe
	probe := builder.getLivenessProbe()

	// Assert custom values
	require.NotNil(t, probe)
	assert.Equal(t, int32(900), probe.InitialDelaySeconds)
	assert.Equal(t, int32(10), probe.TimeoutSeconds)
	assert.Equal(t, int32(15), probe.PeriodSeconds)
	assert.Equal(t, int32(5), probe.FailureThreshold)

	// Assert custom HTTPGet configuration
	require.NotNil(t, probe.HTTPGet)
	assert.Equal(t, "127.0.0.1", probe.HTTPGet.Host)
	assert.Equal(t, "/health", probe.HTTPGet.Path)
	assert.Equal(t, intstr.FromInt32(42699), probe.HTTPGet.Port)
}

func TestGetLivenessProbe_PartialCustomValues(t *testing.T) {
	// Create a custom liveness probe with only some fields set
	customProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Host: "127.0.0.1",
				Path: "/status",
				Port: intstr.FromInt32(ports.InstanaAgentAPIPortConfig.Port),
			},
		},
		InitialDelaySeconds: 1200,
		// Other fields will use zero values
	}

	// Create an InstanaAgent with partial custom liveness probe
	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
		},
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				EndpointHost: "ingress-red-saas.instana.io",
				EndpointPort: "443",
				Key:          "test-key",
				Pod: instanav1.AgentPodSpec{
					LivenessProbe: customProbe,
				},
			},
			Cluster: instanav1.Name{
				Name: "test-cluster",
			},
		},
	}

	// Apply defaults
	agent.Default()

	// Create the builder
	mockClient := &mocks.MockInstanaAgentClient{}
	eventRecorder := record.NewFakeRecorder(10)
	statusManager := status.NewAgentStatusManager(mockClient, eventRecorder)
	builder := NewDaemonSetBuilder(agent, false, statusManager).(*daemonSetBuilder)

	// Get the liveness probe
	probe := builder.getLivenessProbe()

	// Assert that the custom probe is returned as-is
	require.NotNil(t, probe)
	assert.Equal(t, int32(1200), probe.InitialDelaySeconds)
	assert.Equal(t, int32(0), probe.TimeoutSeconds)   // Zero value since not set
	assert.Equal(t, int32(0), probe.PeriodSeconds)    // Zero value since not set
	assert.Equal(t, int32(0), probe.FailureThreshold) // Zero value since not set
}

// theoretically should not be used with the agent, but the kubernetes spec would allow to define it, so adding tests
func TestGetLivenessProbe_TCPSocket(t *testing.T) {
	// Create a custom liveness probe using TCP socket instead of HTTP
	customProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.FromInt32(42699),
			},
		},
		InitialDelaySeconds: 300,
		TimeoutSeconds:      3,
		PeriodSeconds:       5,
		FailureThreshold:    2,
	}

	// Create an InstanaAgent with TCP socket liveness probe
	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
		},
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				EndpointHost: "ingress-red-saas.instana.io",
				EndpointPort: "443",
				Key:          "test-key",
				Pod: instanav1.AgentPodSpec{
					LivenessProbe: customProbe,
				},
			},
			Cluster: instanav1.Name{
				Name: "test-cluster",
			},
		},
	}

	// Apply defaults
	agent.Default()

	// Create the builder
	mockClient := &mocks.MockInstanaAgentClient{}
	eventRecorder := record.NewFakeRecorder(10)
	statusManager := status.NewAgentStatusManager(mockClient, eventRecorder)
	builder := NewDaemonSetBuilder(agent, false, statusManager).(*daemonSetBuilder)

	// Get the liveness probe
	probe := builder.getLivenessProbe()

	// Assert TCP socket configuration
	require.NotNil(t, probe)
	require.NotNil(t, probe.TCPSocket)
	assert.Nil(t, probe.HTTPGet) // HTTPGet should be nil when using TCPSocket
	assert.Equal(t, intstr.FromInt32(42699), probe.TCPSocket.Port)
	assert.Equal(t, int32(300), probe.InitialDelaySeconds)
	assert.Equal(t, int32(3), probe.TimeoutSeconds)
	assert.Equal(t, int32(5), probe.PeriodSeconds)
	assert.Equal(t, int32(2), probe.FailureThreshold)
}

// theoretically should not be used with the agent, but the kubernetes spec would allow to define it, so adding tests
func TestGetLivenessProbe_ExecAction(t *testing.T) {
	// Create a custom liveness probe using Exec action
	customProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: []string{"/bin/sh", "-c", "curl -f http://localhost:42699/status"},
			},
		},
		InitialDelaySeconds: 450,
		TimeoutSeconds:      8,
		PeriodSeconds:       20,
		FailureThreshold:    4,
	}

	// Create an InstanaAgent with Exec liveness probe
	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
		},
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				EndpointHost: "ingress-red-saas.instana.io",
				EndpointPort: "443",
				Key:          "test-key",
				Pod: instanav1.AgentPodSpec{
					LivenessProbe: customProbe,
				},
			},
			Cluster: instanav1.Name{
				Name: "test-cluster",
			},
		},
	}

	// Apply defaults
	agent.Default()

	// Create the builder
	mockClient := &mocks.MockInstanaAgentClient{}
	eventRecorder := record.NewFakeRecorder(10)
	statusManager := status.NewAgentStatusManager(mockClient, eventRecorder)
	builder := NewDaemonSetBuilder(agent, false, statusManager).(*daemonSetBuilder)

	// Get the liveness probe
	probe := builder.getLivenessProbe()

	// Assert Exec configuration
	require.NotNil(t, probe)
	require.NotNil(t, probe.Exec)
	assert.Nil(t, probe.HTTPGet) // HTTPGet should be nil when using Exec
	assert.Equal(
		t,
		[]string{"/bin/sh", "-c", "curl -f http://localhost:42699/status"},
		probe.Exec.Command,
	)
	assert.Equal(t, int32(450), probe.InitialDelaySeconds)
	assert.Equal(t, int32(8), probe.TimeoutSeconds)
	assert.Equal(t, int32(20), probe.PeriodSeconds)
	assert.Equal(t, int32(4), probe.FailureThreshold)
}

func TestBuild_LivenessProbeInDaemonSet(t *testing.T) {
	// Create an InstanaAgent with custom liveness probe
	customProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Host: "127.0.0.1",
				Path: "/status",
				Port: intstr.FromInt32(ports.InstanaAgentAPIPortConfig.Port),
			},
		},
		InitialDelaySeconds: 800,
		TimeoutSeconds:      7,
		PeriodSeconds:       12,
		FailureThreshold:    4,
	}

	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
		},
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				EndpointHost: "ingress-red-saas.instana.io",
				EndpointPort: "443",
				Key:          "test-key",
				Pod: instanav1.AgentPodSpec{
					LivenessProbe: customProbe,
				},
			},
			Cluster: instanav1.Name{
				Name: "test-cluster",
			},
		},
	}

	// Apply defaults
	agent.Default()

	// Create the builder
	mockClient := &mocks.MockInstanaAgentClient{}
	eventRecorder := record.NewFakeRecorder(10)
	statusManager := status.NewAgentStatusManager(mockClient, eventRecorder)
	builder := NewDaemonSetBuilder(agent, false, statusManager).(*daemonSetBuilder)

	// Build the DaemonSet
	ds := builder.build()

	// Assert that the DaemonSet contains the custom liveness probe
	require.NotNil(t, ds)
	require.Len(t, ds.Spec.Template.Spec.Containers, 1)

	container := ds.Spec.Template.Spec.Containers[0]
	require.NotNil(t, container.LivenessProbe)
	assert.Equal(t, int32(800), container.LivenessProbe.InitialDelaySeconds)
	assert.Equal(t, int32(7), container.LivenessProbe.TimeoutSeconds)
	assert.Equal(t, int32(12), container.LivenessProbe.PeriodSeconds)
	assert.Equal(t, int32(4), container.LivenessProbe.FailureThreshold)
}

func TestBuild_DefaultLivenessProbeInDaemonSet(t *testing.T) {
	// Create an InstanaAgent without custom liveness probe
	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
		},
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				EndpointHost: "ingress-red-saas.instana.io",
				EndpointPort: "443",
				Key:          "test-key",
			},
			Cluster: instanav1.Name{
				Name: "test-cluster",
			},
		},
	}

	// Apply defaults
	agent.Default()

	// Create the builder
	mockClient := &mocks.MockInstanaAgentClient{}
	eventRecorder := record.NewFakeRecorder(10)
	statusManager := status.NewAgentStatusManager(mockClient, eventRecorder)
	builder := NewDaemonSetBuilder(agent, false, statusManager).(*daemonSetBuilder)

	// Build the DaemonSet
	ds := builder.build()

	// Assert that the DaemonSet contains the default liveness probe
	require.NotNil(t, ds)
	require.Len(t, ds.Spec.Template.Spec.Containers, 1)

	container := ds.Spec.Template.Spec.Containers[0]
	require.NotNil(t, container.LivenessProbe)
	assert.Equal(t, int32(600), container.LivenessProbe.InitialDelaySeconds)
	assert.Equal(t, int32(5), container.LivenessProbe.TimeoutSeconds)
	assert.Equal(t, int32(10), container.LivenessProbe.PeriodSeconds)
	assert.Equal(t, int32(3), container.LivenessProbe.FailureThreshold)
}

func TestGetLivenessProbe_NilPointer(t *testing.T) {
	// Create an InstanaAgent with explicitly nil liveness probe
	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
		},
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				EndpointHost: "ingress-red-saas.instana.io",
				EndpointPort: "443",
				Key:          "test-key",
				Pod: instanav1.AgentPodSpec{
					LivenessProbe: nil,
				},
			},
			Cluster: instanav1.Name{
				Name: "test-cluster",
			},
		},
	}

	// Apply defaults
	agent.Default()

	// Create the builder
	mockClient := &mocks.MockInstanaAgentClient{}
	eventRecorder := record.NewFakeRecorder(10)
	statusManager := status.NewAgentStatusManager(mockClient, eventRecorder)
	builder := NewDaemonSetBuilder(agent, false, statusManager).(*daemonSetBuilder)

	// Get the liveness probe
	probe := builder.getLivenessProbe()

	// Assert default values are returned
	require.NotNil(t, probe)
	assert.Equal(t, int32(600), probe.InitialDelaySeconds)
	assert.Equal(t, int32(5), probe.TimeoutSeconds)
	assert.Equal(t, int32(10), probe.PeriodSeconds)
	assert.Equal(t, int32(3), probe.FailureThreshold)
}

func TestGetLivenessProbe_WithSuccessThreshold(t *testing.T) {
	// Create a custom liveness probe with SuccessThreshold set
	customProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Host: "127.0.0.1",
				Path: "/status",
				Port: intstr.FromInt32(ports.InstanaAgentAPIPortConfig.Port),
			},
		},
		InitialDelaySeconds: 600,
		TimeoutSeconds:      5,
		PeriodSeconds:       10,
		FailureThreshold:    3,
		SuccessThreshold:    2, // Custom success threshold
	}

	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
		},
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				EndpointHost: "ingress-red-saas.instana.io",
				EndpointPort: "443",
				Key:          "test-key",
				Pod: instanav1.AgentPodSpec{
					LivenessProbe: customProbe,
				},
			},
			Cluster: instanav1.Name{
				Name: "test-cluster",
			},
		},
	}

	// Apply defaults
	agent.Default()

	// Create the builder
	mockClient := &mocks.MockInstanaAgentClient{}
	eventRecorder := record.NewFakeRecorder(10)
	statusManager := status.NewAgentStatusManager(mockClient, eventRecorder)
	builder := NewDaemonSetBuilder(agent, false, statusManager).(*daemonSetBuilder)

	// Get the liveness probe
	probe := builder.getLivenessProbe()

	// Assert that SuccessThreshold is preserved
	require.NotNil(t, probe)
	assert.Equal(t, int32(2), probe.SuccessThreshold)
}

// Made with Bob
