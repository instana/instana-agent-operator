package transformations

import (
	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/pointer"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"

	"github.com/instana/instana-agent-operator/pkg/optional"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	version = optional.Of(os.Getenv("OPERATOR_VERSION")).GetOrElse("v0.0.0")
)

type Transformations interface {
	AddCommonLabels(obj client.Object)
	AddOwnerReference(obj client.Object)
}

type transformations struct {
	v1.OwnerReference
}

func NewTransformations(agent *instanav1.InstanaAgent) Transformations {
	return &transformations{
		OwnerReference: v1.OwnerReference{
			APIVersion:         agent.APIVersion,
			Kind:               agent.Kind,
			Name:               agent.Name,
			UID:                agent.UID,
			Controller:         pointer.ToPointer(true),
			BlockOwnerDeletion: pointer.ToPointer(true),
		},
	}
}

func (t *transformations) AddCommonLabels(obj client.Object) {

	labels := optional.Of(obj.GetLabels()).GetOrElse(make(map[string]string, 3))

	labels["app.kubernetes.io/name"] = "instana-agent"
	labels["app.kubernetes.io/instance"] = t.Name
	labels["app.kubernetes.io/version"] = version

	obj.SetLabels(labels)
}

func (t *transformations) AddOwnerReference(obj client.Object) {
	obj.SetOwnerReferences(append(obj.GetOwnerReferences(), t.OwnerReference))
} // TODO: test
