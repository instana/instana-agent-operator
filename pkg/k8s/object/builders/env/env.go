package env

import (
	"github.com/instana/instana-agent-operator/pkg/optional"
	corev1 "k8s.io/api/core/v1"
)

type EnvBuilder interface {
	Build() optional.Optional[corev1.EnvVar]
}
