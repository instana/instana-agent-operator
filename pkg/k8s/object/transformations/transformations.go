package transformations

import (
	"os"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/pointer"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/instana/instana-agent-operator/pkg/optional"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO: Support nameOverride and fullNameOverride ???

var (
	version = optional.Of(os.Getenv("OPERATOR_VERSION")).GetOrDefault("v0.0.0")
)

// labels
const (
	NameLabel     = "app.kubernetes.io/name"
	InstanceLabel = "app.kubernetes.io/instance"
	VersionLabel  = "app.kubernetes.io/version"
)

type Transformations interface {
	AddCommonLabels(obj client.Object)
	AddOwnerReference(obj client.Object)
	AddCommonLabelsToMap(labels map[string]string, name string, skipVersionLabel bool) map[string]string
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
			Controller:         pointer.To(true),
			BlockOwnerDeletion: pointer.To(true),
		},
	}
}

func (t *transformations) AddCommonLabelsToMap(labels map[string]string, name string, skipVersionLabel bool) map[string]string {
	labels[NameLabel] = "instana-agent"
	labels[InstanceLabel] = name
	if !skipVersionLabel {
		labels[VersionLabel] = version
	}
	return labels
}

func (t *transformations) AddCommonLabels(obj client.Object) {
	labels := optional.Of(obj.GetLabels()).GetOrDefault(make(map[string]string, 3))
	t.AddCommonLabelsToMap(labels, t.Name, false)
	obj.SetLabels(labels)
}

func (t *transformations) AddOwnerReference(obj client.Object) {
	obj.SetOwnerReferences(append(obj.GetOwnerReferences(), t.OwnerReference)) // TODO: Use contorllerutils function, what to do about cluster-scoped resources?
}
