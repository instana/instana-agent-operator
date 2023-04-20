package env

import (
	corev1 "k8s.io/api/core/v1"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/collections/list"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/helpers"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

// TODO: Test

type EnvVar int

const (
	AgentModeEnv EnvVar = iota // TODO: Rest
)

type EnvBuilder interface {
	Build(envVars ...EnvVar) []corev1.EnvVar
}

type envBuilder struct {
	agent *instanav1.InstanaAgent
	helpers.Helpers

	builders map[EnvVar]func() optional.Optional[corev1.EnvVar]
}

func (e *envBuilder) Build(envVars ...EnvVar) []corev1.EnvVar {
	optionals := list.NewListMapTo[EnvVar, optional.Optional[corev1.EnvVar]]().MapTo(
		envVars,
		func(envVar EnvVar) optional.Optional[corev1.EnvVar] {
			return e.builders[envVar]()
		},
	)

	return optional.NewNonEmptyOptionalMapper[corev1.EnvVar]().AllNonEmpty(optionals)
}

// TODO: Need to ensure no risk of overlooking panic in production if additional vars are added later, possibly use exhaustive linter?
func (e *envBuilder) initializeBuildersMap() {
	e.builders = map[EnvVar]func() optional.Optional[corev1.EnvVar]{
		AgentModeEnv: e.agentModeEnv,
	}
}

func NewEnvBuilder(agent *instanav1.InstanaAgent) EnvBuilder {
	res := &envBuilder{
		agent:   agent,
		Helpers: helpers.NewHelpers(agent),
	}

	res.initializeBuildersMap()

	return res
}
