package rbac

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/builder"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

type rbacBaseBuilder struct {
	*instanav1.InstanaAgent
	isOpenShift bool
	builder.ObjectBuilder
}

func (r *rbacBaseBuilder) Build() optional.Optional[client.Object] {
	switch r.isOpenShift || r.Spec.Rbac.Create {
	case true:
		return r.ObjectBuilder.Build()
	default:
		return optional.Empty[client.Object]()
	}
}

func newRbacBuilder(
	agent *instanav1.InstanaAgent,
	isOpenShift bool,
	builder builder.ObjectBuilder,
) builder.ObjectBuilder {
	return &rbacBaseBuilder{
		InstanaAgent:  agent,
		isOpenShift:   isOpenShift,
		ObjectBuilder: builder,
	}
}
