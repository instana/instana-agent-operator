package env

import (
	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/optional"
	corev1 "k8s.io/api/core/v1"
)

type agentMode struct {
	*instanav1.InstanaAgent
}

func AgentModeEnv(agent *instanav1.InstanaAgent) EnvBuilder {
	return &agentMode{
		InstanaAgent: agent,
	}
}

// TODO: Generalize this logic for other similar cases

func (a *agentMode) Build() optional.Optional[corev1.EnvVar] {
	switch mode := a.Spec.Agent.Mode; mode == "" {
	case true:
		return optional.Empty[corev1.EnvVar]()
	default:
		return optional.Of(corev1.EnvVar{
			Name:  "INSTANA_AGENT_MODE",
			Value: string(mode),
		})
	}
} // TODO: Test and incorporate
