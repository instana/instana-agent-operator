package helpers

import (
	v1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

var (
	instance = helpers{}
)

type helpers struct{}

type Helpers interface {
	ServiceAccountName(agent *v1.InstanaAgent) string
}

func (h *helpers) serviceAccountNameDefault(agent *v1.InstanaAgent) string {
	switch agent.Spec.ServiceAccountSpec.Create.Create {
	case true:
		return agent.Name
	default:
		return "default"
	}
}

func (h *helpers) ServiceAccountName(agent *v1.InstanaAgent) string {
	return optional.Of(agent.Spec.ServiceAccountSpec.Name.Name).GetOrElse(h.serviceAccountNameDefault(agent))
}

func GetInstance() Helpers {
	return &instance
}
