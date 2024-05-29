package env

import (
	"os"

	"github.com/instana/instana-agent-operator/pkg/optional"
)

var (
	operatorVersion = optional.Of(os.Getenv("OPERATOR_VERSION")).GetOrDefault("v0.0.1-dev")
)

func GetOperatorVersion() string {
	return operatorVersion
}
