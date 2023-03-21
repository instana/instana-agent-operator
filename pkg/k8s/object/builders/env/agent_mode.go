package env

import (
	instanav1 "github.com/instana/instana-agent-operator/api/v1"
)

func AgentModeEnv(agent *instanav1.InstanaAgent) EnvBuilder {
	return fromField("INSTANA_AGENT_MODE", agent.Spec.Agent.Mode)
}
