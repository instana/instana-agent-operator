package transformations

import (
	"os"

	"github.com/instana/instana-agent-operator/pkg/optional"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	version = optional.Of(os.Getenv("OPERATOR_VERSION")).GetOrElse("v0.0.0")
)

// TODO: interface this for mocking

func AddCommonLabels(obj client.Object) {

	labels := optional.Of(obj.GetLabels()).GetOrElse(make(map[string]string, 3))

	labels["app.kubernetes.io/name"] = "instana-agent"
	labels["app.kubernetes.io/instance"] = "instana-agent"
	labels["app.kubernetes.io/version"] = version

	obj.SetLabels(labels)
}
