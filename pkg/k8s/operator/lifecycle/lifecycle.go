package lifecycle

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/collections/list"
	"github.com/instana/instana-agent-operator/pkg/json_or_die"
	instanaclient "github.com/instana/instana-agent-operator/pkg/k8s/client"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/transformations"
	"github.com/instana/instana-agent-operator/pkg/multierror"
	"github.com/instana/instana-agent-operator/pkg/result"
)

// TODO: Test

type DependentLifecycleManager interface {
	UpdateDependentLifecycleInfo(currentGenerationDependents []client.Object) instanaclient.MultiObjectResult
	DeleteOrphanedDependents(currentGenerationDependents []client.Object) instanaclient.MultiObjectResult
	DeleteAllDependents() instanaclient.MultiObjectResult
}

type dependentLifecycleManager struct {
	ctx   context.Context
	agent client.Object

	instanaclient.InstanaAgentClient
	objectStrip
	json_or_die.JsonOrDieMarshaler[[]unstructured.Unstructured]
	transformations.Transformations
}

func (d *dependentLifecycleManager) getCmName() string {
	return d.agent.GetName() + "-dependents"
}

func (d *dependentLifecycleManager) toStripped(objects []client.Object) []unstructured.Unstructured {
	return list.NewListMapTo[client.Object, unstructured.Unstructured]().MapTo(
		objects,
		d.stripObject,
	)
}

func (d *dependentLifecycleManager) marshalDependents(currentGenerationDependents []client.Object) []byte {
	return d.MarshalOrDie(d.toStripped(currentGenerationDependents))
}

func (d *dependentLifecycleManager) getCurrentGenKey() string {
	return fmt.Sprintf("%s_%d", transformations.GetVersion(), d.agent.GetGeneration())
}

func (d *dependentLifecycleManager) UpdateDependentLifecycleInfo(currentGenerationDependents []client.Object) instanaclient.MultiObjectResult {
	// TODO: Need to pull in existing keys first

	lifecycleCm := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.getCmName(),
			Namespace: d.agent.GetNamespace(),
		},
		Data: map[string]string{
			d.getCurrentGenKey(): string(d.marshalDependents(currentGenerationDependents)),
		},
	}

	d.AddCommonLabels(&lifecycleCm, constants.ComponentInstanaAgent)
	d.AddOwnerReference(&lifecycleCm)

	_, err := d.Apply(d.ctx, &lifecycleCm).Get()
	return result.Of(currentGenerationDependents, err)
}

func (d *dependentLifecycleManager) getLifecycleCm() result.Result[corev1.ConfigMap] {
	lifecycleCm := corev1.ConfigMap{}

	return result.Of(
		lifecycleCm,
		d.Get(d.ctx, types.NamespacedName{Name: d.getCmName(), Namespace: d.agent.GetNamespace()}, &lifecycleCm),
	)
}

func (d *dependentLifecycleManager) getGeneration(
	lifecycleCm *corev1.ConfigMap,
	key string,
) []unstructured.Unstructured {
	return result.OfInlineCatchingPanic[[]unstructured.Unstructured](
		func() (res []unstructured.Unstructured, err error) {
			return d.JsonOrDieMarshaler.UnMarshalOrDie([]byte(lifecycleCm.Data[key])), nil
		},
	).ToOptional().GetOrElse(
		func() []unstructured.Unstructured {
			return make([]unstructured.Unstructured, 0)
		},
	)
}

func (d *dependentLifecycleManager) deleteAll(toDelete []unstructured.Unstructured) result.Result[[]client.Object] {
	toDeleteCasted := list.NewListMapTo[unstructured.Unstructured, client.Object]().MapTo(
		toDelete,
		func(val unstructured.Unstructured) client.Object {
			return &val
		},
	)

	return d.DeleteAllInTimeLimit(d.ctx, toDeleteCasted, 30*time.Second, 5*time.Second)
}

func (d *dependentLifecycleManager) deleteOrphanedDependents(
	lifecycleCm *corev1.ConfigMap,
	currentGenerationDependents []client.Object,
) result.Result[[]client.Object] {
	errBuilder := multierror.NewMultiErrorBuilder()

	currentGeneration := d.toStripped(currentGenerationDependents)

	for key := range lifecycleCm.Data {
		olderGeneration := d.getGeneration(lifecycleCm, key)
		deprecatedDependents := list.NewDeepDiff[unstructured.Unstructured]().Diff(
			olderGeneration,
			currentGeneration,
		)
		d.deleteAll(deprecatedDependents).
			OnSuccess(
				func(_ []client.Object) {
					if key != d.getCurrentGenKey() {
						delete(lifecycleCm.Data, key)
					}
				},
			).
			OnFailure(errBuilder.AddSingle)
	}

	d.Apply(d.ctx, lifecycleCm).OnFailure(errBuilder.AddSingle)

	return result.Of(currentGenerationDependents, errBuilder.Build())
}

func (d *dependentLifecycleManager) DeleteOrphanedDependents(currentGenerationDependents []client.Object) instanaclient.MultiObjectResult {
	return result.Map[corev1.ConfigMap, []client.Object](
		d.getLifecycleCm(),
		func(lifecycleCm corev1.ConfigMap) result.Result[[]client.Object] {
			return d.deleteOrphanedDependents(&lifecycleCm, currentGenerationDependents)
		},
	)
}

func (d *dependentLifecycleManager) DeleteAllDependents() instanaclient.MultiObjectResult {
	return d.DeleteOrphanedDependents(nil)
}

func NewDependentLifecycleManager(
	ctx context.Context,
	agent *instanav1.InstanaAgent,
	instanaClient instanaclient.InstanaAgentClient,
) DependentLifecycleManager {
	return &dependentLifecycleManager{
		ctx:   ctx,
		agent: agent,

		InstanaAgentClient: instanaClient,
		objectStrip:        &strip{},
		JsonOrDieMarshaler: json_or_die.NewJsonOrDieArray[unstructured.Unstructured](),
		Transformations:    transformations.NewTransformations(agent),
	}
}
