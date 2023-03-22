package env

import (
	instanav1 "github.com/instana/instana-agent-operator/api/v1"
)

func AgentModeEnv(agent *instanav1.InstanaAgent) EnvBuilder {
	return fromField("INSTANA_AGENT_MODE", agent.Spec.Agent.Mode)
}

// TODO: Leader elector port still needed?

func ZoneNameEnv(agent *instanav1.InstanaAgent) EnvBuilder {
	return fromField("INSTANA_ZONE", agent.Spec.Zone.Name)
}

func ClusterNameEnv(agent *instanav1.InstanaAgent) EnvBuilder {
	return fromField("INSTANA_KUBERNETES_CLUSTER_NAME", agent.Spec.Cluster.Name)
}

func AgentEndpointEnv(agent *instanav1.InstanaAgent) EnvBuilder {
	return fromField("INSTANA_AGENT_ENDPOINT", agent.Spec.Agent.EndpointHost)
}

func AgentEndpointPortEnv(agent *instanav1.InstanaAgent) EnvBuilder {
	return fromField("INSTANA_AGENT_ENDPOINT_PORT", agent.Spec.Agent.EndpointPort)
}
