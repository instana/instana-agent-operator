package helpers

import (
	v1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

// TODO: Mockable

func serviceAccountNameDefault(agent *v1.InstanaAgent) string {
	switch agent.Spec.ServiceAccountSpec.Create.Create {
	case true:
		return agent.Name
	default:
		return "default"
	}
}

func ServiceAccountName(agent *v1.InstanaAgent) string {
	return optional.Of(agent.Spec.ServiceAccountSpec.Name.Name).GetOrElse(serviceAccountNameDefault(agent))
}
