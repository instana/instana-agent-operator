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
	"github.com/instana/instana-agent-operator/pkg/k8s/object/transformations"
	"github.com/instana/instana-agent-operator/pkg/multierror"
	"github.com/instana/instana-agent-operator/pkg/result"
)

// TODO: +Delete All version for finalizer logic

// TODO: Test

type DependentLifecycleManager interface {
	UpdateDependentLifecycleInfo() result.Result[corev1.ConfigMap]
	DeleteOrphanedDependents() result.Result[corev1.ConfigMap]
}

type dependentLifecycleManager struct {
	ctx                         context.Context
	agent                       client.Object
	currentGenerationDependents []client.Object

	instanaclient.InstanaAgentClient
	objectStrip
	json_or_die.JsonOrDieMarshaler[[]unstructured.Unstructured]
}

func (d *dependentLifecycleManager) getCmName() string {
	return d.agent.GetName() + "-dependents"
}

func (d *dependentLifecycleManager) marshalDependents() []byte {
	stripped := list.NewListMapTo[client.Object, unstructured.Unstructured]().MapTo(
		d.currentGenerationDependents,
		d.stripObject,
	)

	return d.MarshalOrDie(stripped)
}

func (d *dependentLifecycleManager) getCurrentGenKey() string {
	return fmt.Sprintf("%s_%d", transformations.GetVersion(), d.agent.GetGeneration())
}

func (d *dependentLifecycleManager) UpdateDependentLifecycleInfo() result.Result[corev1.ConfigMap] {
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
			d.getCurrentGenKey(): string(d.marshalDependents()),
		},
	}

	_, err := d.Apply(d.ctx, &lifecycleCm).Get()
	return result.Of(lifecycleCm, err)
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

func (d *dependentLifecycleManager) deleteOrphanedDependents(lifecycleCm *corev1.ConfigMap) result.Result[corev1.ConfigMap] {
	errBuilder := multierror.NewMultiErrorBuilder()

	currentGeneration := d.getGeneration(lifecycleCm, d.getCurrentGenKey())

	for key := range lifecycleCm.Data {
		olderGeneration := d.getGeneration(lifecycleCm, key)
		deprecatedDependents := list.NewDeepDiff[unstructured.Unstructured]().Diff(
			olderGeneration,
			currentGeneration,
		)
		d.deleteAll(deprecatedDependents).
			OnSuccess(
				func(_ []client.Object) {
					delete(lifecycleCm.Data, key)
				},
			).
			OnFailure(errBuilder.AddSingle)
	}

	d.Apply(d.ctx, lifecycleCm).OnFailure(errBuilder.AddSingle)

	return result.Of(*lifecycleCm, errBuilder.Build())
}

func (d *dependentLifecycleManager) DeleteOrphanedDependents() result.Result[corev1.ConfigMap] {
	return result.Map[corev1.ConfigMap, corev1.ConfigMap](
		d.getLifecycleCm(),
		func(lifecycleCm corev1.ConfigMap) result.Result[corev1.ConfigMap] {
			return d.deleteOrphanedDependents(&lifecycleCm)
		},
	)
}

func NewDependentLifecycleManager(
	ctx context.Context,
	agent *instanav1.InstanaAgent,
	currentGenerationDependents []client.Object,
	instanaClient instanaclient.InstanaAgentClient,
) DependentLifecycleManager {
	return &dependentLifecycleManager{
		ctx:                         ctx,
		agent:                       agent,
		currentGenerationDependents: currentGenerationDependents,

		InstanaAgentClient: instanaClient,
		objectStrip:        &strip{},
		JsonOrDieMarshaler: json_or_die.NewJsonOrDieArray[unstructured.Unstructured](),
	}
}
