package transformations

import (
	"os"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/or_die"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

// TODO: Track list of cluster-scoped and namespace-scoped dependents (to cleanup deprecated resources) + Forbid Create/Update/Patch if unregistered (possibly use runtime.Scheme for this)

// labels
const (
	NameLabel       = "app.kubernetes.io/name"
	InstanceLabel   = "app.kubernetes.io/instance"
	VersionLabel    = "app.kubernetes.io/version"
	ComponentLabel  = "app.kubernetes.io/component"
	PartOfLabel     = "app.kubernetes.io/part-of"
	ManagedByLabel  = "app.kubernetes.io/managed-by"
	GenerationLabel = "agent.instana.io/generation"
)

const (
	name      = "instana-agent"
	partOf    = "instana"
	managedBy = "instana-agent-operator"
)

var (
	version = optional.Of(os.Getenv("OPERATOR_VERSION")).GetOrDefault("v0.0.0")
)

type Transformations interface {
	AddCommonLabels(obj client.Object)
	AddOwnerReference(obj client.Object)
	PreviousGenerationsSelector() labels.Selector
}

type transformations struct {
	metav1.OwnerReference
	generation string
	component  string
}

func (t *transformations) AddCommonLabels(obj client.Object) {
	objLabels := optional.Of(obj.GetLabels()).GetOrDefault(make(map[string]string, 7))

	objLabels[NameLabel] = name
	objLabels[InstanceLabel] = t.Name
	objLabels[VersionLabel] = version
	objLabels[ComponentLabel] = t.component
	objLabels[PartOfLabel] = partOf
	objLabels[ManagedByLabel] = managedBy
	objLabels[GenerationLabel] = t.generation

	obj.SetLabels(objLabels)
}

// TODO: Test

func (t *transformations) PreviousGenerationsSelector() labels.Selector {
	return or_die.New[labels.Selector]().ResultOrDie(
		func() (labels.Selector, error) {
			return metav1.LabelSelectorAsSelector(
				&metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      NameLabel,
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{name},
						},
						{
							Key:      InstanceLabel,
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{t.Name},
						},
						{
							Key:      GenerationLabel,
							Operator: metav1.LabelSelectorOpNotIn,
							Values:   []string{t.generation},
						},
					},
				},
			)
		},
	)
}

func (t *transformations) AddOwnerReference(obj client.Object) {
	obj.SetOwnerReferences(
		append(
			obj.GetOwnerReferences(),
			t.OwnerReference,
		),
	) // TODO: cluster-scoped resources
}

func NewTransformations(agent *instanav1.InstanaAgent, component string) Transformations {
	return &transformations{
		OwnerReference: metav1.OwnerReference{
			APIVersion:         agent.APIVersion,
			Kind:               agent.Kind,
			Name:               agent.Name,
			UID:                agent.UID,
			Controller:         pointer.To(true),
			BlockOwnerDeletion: pointer.To(true),
		},
		generation: version + "-" + strconv.Itoa(int(agent.Generation)),
		component:  component,
	}
}
