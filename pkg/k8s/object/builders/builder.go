package builders

import (
	"github.com/instana/instana-agent-operator/pkg/optional"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ResourceBuilder interface {
	optional.Builder[client.Object]
}
