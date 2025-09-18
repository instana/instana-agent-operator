package env

import (
	"testing"

	"github.com/stretchr/testify/assert"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

func TestEnvBuilder_ETCDCAFileEnv(t *testing.T) {
	// Given
	agent := &instanav1.InstanaAgent{
		Spec: instanav1.InstanaAgentSpec{
			K8sSensor: instanav1.K8sSpec{
				ETCD: instanav1.ETCDSpec{
					CA: instanav1.CASpec{
						MountPath: "/etc/ssl/certs",
					},
				},
			},
		},
	}
	builder := NewEnvBuilder(agent, nil)

	// When
	envVars := builder.Build(ETCDCAFileEnv)

	// Then
	assert.Len(t, envVars, 1, "Should create one env var")
	assert.Equal(t, "ETCD_CA_FILE", envVars[0].Name, "Env var name should be ETCD_CA_FILE")
	assert.Equal(
		t,
		"/etc/ssl/certs/ca.crt",
		envVars[0].Value,
		"Env var value should be path to CA file",
	)
}

func TestEnvBuilder_ETCDCAFileEnv_NoMountPath(t *testing.T) {
	// Given
	agent := &instanav1.InstanaAgent{
		Spec: instanav1.InstanaAgentSpec{
			K8sSensor: instanav1.K8sSpec{
				ETCD: instanav1.ETCDSpec{
					CA: instanav1.CASpec{
						// No MountPath specified
					},
				},
			},
		},
	}
	builder := NewEnvBuilder(agent, nil)

	// When
	envVars := builder.Build(ETCDCAFileEnv)

	// Then
	assert.Len(t, envVars, 0, "Should not create any env vars")
}

func TestEnvBuilder_ETCDInsecureEnv_True(t *testing.T) {
	// Given
	agent := &instanav1.InstanaAgent{
		Spec: instanav1.InstanaAgentSpec{
			K8sSensor: instanav1.K8sSpec{
				ETCD: instanav1.ETCDSpec{
					Insecure: pointer.To(true),
				},
			},
		},
	}
	builder := NewEnvBuilder(agent, nil)

	// When
	envVars := builder.Build(ETCDInsecureEnv)

	// Then
	assert.Len(t, envVars, 1, "Should create one env var")
	assert.Equal(t, "ETCD_INSECURE", envVars[0].Name, "Env var name should be ETCD_INSECURE")
	assert.Equal(t, "true", envVars[0].Value, "Env var value should be true")
}

func TestEnvBuilder_ETCDInsecureEnv_False(t *testing.T) {
	// Given
	agent := &instanav1.InstanaAgent{
		Spec: instanav1.InstanaAgentSpec{
			K8sSensor: instanav1.K8sSpec{
				ETCD: instanav1.ETCDSpec{
					Insecure: pointer.To(false),
				},
			},
		},
	}
	builder := NewEnvBuilder(agent, nil)

	// When
	envVars := builder.Build(ETCDInsecureEnv)

	// Then
	assert.Len(t, envVars, 1, "Should create one env var")
	assert.Equal(t, "ETCD_INSECURE", envVars[0].Name, "Env var name should be ETCD_INSECURE")
	assert.Equal(t, "false", envVars[0].Value, "Env var value should be false")
}

func TestEnvBuilder_ETCDInsecureEnv_NotSpecified(t *testing.T) {
	// Given
	agent := &instanav1.InstanaAgent{
		Spec: instanav1.InstanaAgentSpec{
			K8sSensor: instanav1.K8sSpec{
				ETCD: instanav1.ETCDSpec{
					// Insecure not specified
				},
			},
		},
	}
	builder := NewEnvBuilder(agent, nil)

	// When
	envVars := builder.Build(ETCDInsecureEnv)

	// Then
	assert.Len(t, envVars, 0, "Should not create any env vars")
}

func TestEnvBuilder_ETCDTargetsEnv(t *testing.T) {
	// Given
	agent := &instanav1.InstanaAgent{
		Spec: instanav1.InstanaAgentSpec{
			K8sSensor: instanav1.K8sSpec{
				ETCD: instanav1.ETCDSpec{
					Targets: []string{"https://etcd-1:2379", "https://etcd-2:2379"},
				},
			},
		},
	}
	builder := NewEnvBuilder(agent, nil)

	// When
	envVars := builder.Build(ETCDTargetsEnv)

	// Then
	assert.Len(t, envVars, 1, "Should create one env var")
	assert.Equal(t, "ETCD_TARGETS", envVars[0].Name, "Env var name should be ETCD_TARGETS")
	assert.Equal(
		t,
		"https://etcd-1:2379,https://etcd-2:2379",
		envVars[0].Value,
		"Env var value should be comma-separated list of targets",
	)
}

func TestEnvBuilder_ETCDTargetsEnv_NoTargets(t *testing.T) {
	// Given
	agent := &instanav1.InstanaAgent{
		Spec: instanav1.InstanaAgentSpec{
			K8sSensor: instanav1.K8sSpec{
				ETCD: instanav1.ETCDSpec{
					// No targets specified
				},
			},
		},
	}
	builder := NewEnvBuilder(agent, nil)

	// When
	envVars := builder.Build(ETCDTargetsEnv)

	// Then
	assert.Len(t, envVars, 0, "Should not create any env vars")
}

func TestEnvBuilder_ControlPlaneCAFileEnv(t *testing.T) {
	// Given
	agent := &instanav1.InstanaAgent{
		Spec: instanav1.InstanaAgentSpec{
			K8sSensor: instanav1.K8sSpec{
				RestClient: instanav1.RestClientSpec{
					CA: instanav1.CASpec{
						MountPath: "/etc/ssl/control-plane",
					},
				},
			},
		},
	}
	builder := NewEnvBuilder(agent, nil)

	// When
	envVars := builder.Build(ControlPlaneCAFileEnv)

	// Then
	assert.Len(t, envVars, 1, "Should create one env var")
	assert.Equal(
		t,
		"CONTROL_PLANE_CA_FILE",
		envVars[0].Name,
		"Env var name should be CONTROL_PLANE_CA_FILE",
	)
	assert.Equal(
		t,
		"/etc/ssl/control-plane/ca.crt",
		envVars[0].Value,
		"Env var value should be path to CA file",
	)
}

func TestEnvBuilder_ControlPlaneCAFileEnv_NoMountPath(t *testing.T) {
	// Given
	agent := &instanav1.InstanaAgent{
		Spec: instanav1.InstanaAgentSpec{
			K8sSensor: instanav1.K8sSpec{
				RestClient: instanav1.RestClientSpec{
					CA: instanav1.CASpec{
						// No MountPath specified
					},
				},
			},
		},
	}
	builder := NewEnvBuilder(agent, nil)

	// When
	envVars := builder.Build(ControlPlaneCAFileEnv)

	// Then
	assert.Len(t, envVars, 0, "Should not create any env vars")
}

func TestEnvBuilder_RestClientHostAllowlistEnv(t *testing.T) {
	// Given
	agent := &instanav1.InstanaAgent{
		Spec: instanav1.InstanaAgentSpec{
			K8sSensor: instanav1.K8sSpec{
				RestClient: instanav1.RestClientSpec{
					HostAllowlist: []string{"localhost", "127.0.0.1", "kubernetes.default.svc"},
				},
			},
		},
	}
	builder := NewEnvBuilder(agent, nil)

	// When
	envVars := builder.Build(RestClientHostAllowlistEnv)

	// Then
	assert.Len(t, envVars, 1, "Should create one env var")
	assert.Equal(
		t,
		"REST_CLIENT_HOST_ALLOWLIST",
		envVars[0].Name,
		"Env var name should be REST_CLIENT_HOST_ALLOWLIST",
	)
	assert.Equal(
		t,
		"localhost,127.0.0.1,kubernetes.default.svc",
		envVars[0].Value,
		"Env var value should be comma-separated list of hosts",
	)
}

func TestEnvBuilder_RestClientHostAllowlistEnv_NoHosts(t *testing.T) {
	// Given
	agent := &instanav1.InstanaAgent{
		Spec: instanav1.InstanaAgentSpec{
			K8sSensor: instanav1.K8sSpec{
				RestClient: instanav1.RestClientSpec{
					// No hosts specified
				},
			},
		},
	}
	builder := NewEnvBuilder(agent, nil)

	// When
	envVars := builder.Build(RestClientHostAllowlistEnv)

	// Then
	assert.Len(t, envVars, 0, "Should not create any env vars")
}
