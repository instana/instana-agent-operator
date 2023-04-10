package transformations

import (
	"os"
	"strconv"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/pointer"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/instana/instana-agent-operator/pkg/optional"
)

// TODO: Support nameOverride and fullNameOverride ???

var (
	version = optional.Of(os.Getenv("OPERATOR_VERSION")).GetOrDefault("v0.0.0")
)

// labels
const (
	NameLabel       = "app.kubernetes.io/name"
	InstanceLabel   = "app.kubernetes.io/instance"
	VersionLabel    = "app.kubernetes.io/version"
	GenerationLabel = "agent.instana.io/generation"
)

type Transformations interface {
	AddCommonLabels(obj client.Object)
	AddOwnerReference(obj client.Object)
	AddCommonLabelsToMap(labels map[string]string, name string, skipVersionLabel bool) map[string]string
}

type transformations struct {
	v1.OwnerReference
	generation string
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
		generation: strconv.Itoa(int(agent.Generation)),
	}
}

func (t *transformations) AddCommonLabelsToMap(
	labels map[string]string,
	name string,
	skipVersionLabel bool,
) map[string]string {
	return t.addCommonLabelsToMap(labels, name, skipVersionLabel, optional.Empty[string]())
}

func (t *transformations) addCommonLabelsToMap(
	labels map[string]string,
	name string,
	skipVersionLabel bool,
	generation optional.Optional[string],
) map[string]string {
	labels[NameLabel] = "instana-agent"
	labels[InstanceLabel] = name
	if !skipVersionLabel {
		labels[VersionLabel] = version
	}
	generation.IfPresent(
		func(gen string) {
			labels[GenerationLabel] = gen
		},
	)
	return labels
}

func (t *transformations) AddCommonLabels(obj client.Object) {
	labels := optional.Of(obj.GetLabels()).GetOrDefault(make(map[string]string, 4))
	t.addCommonLabelsToMap(labels, t.Name, false, optional.Of(t.generation))
	obj.SetLabels(labels)
}

func (t *transformations) AddOwnerReference(obj client.Object) {
	obj.SetOwnerReferences(
		append(
			obj.GetOwnerReferences(),
			t.OwnerReference,
		),
	) // TODO: Use contorllerutils function, what to do about cluster-scoped resources?
}
