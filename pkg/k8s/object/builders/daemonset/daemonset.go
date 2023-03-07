package daemonset

import (
	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DaemonSetBuilder interface {
	Build() optional.Optional[client.Object]
}

type daemonSetBuilder struct {
	instanav1.InstanaAgentSpec
}

func NewDaemonSetBuilder(agent *instanav1.InstanaAgent) DaemonSetBuilder {
	return &daemonSetBuilder{
		InstanaAgentSpec: agent.Spec,
	}
}

func (d *daemonSetBuilder) Build() optional.Optional[client.Object] {
	panic("implement me")
}
