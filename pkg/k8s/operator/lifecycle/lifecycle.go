package lifecycle

import (
	"context"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/collections/list"
	instanaclient "github.com/instana/instana-agent-operator/pkg/k8s/client"
	"github.com/instana/instana-agent-operator/pkg/or_die"
	"github.com/instana/instana-agent-operator/pkg/result"
)

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
	jsonMarshaler
}

func (d *dependentLifecycleManager) getCmName() string {
	return d.agent.GetName() + "-dependents"
}

func (d *dependentLifecycleManager) marshalDependents() []byte {
	stripped := list.NewListMapTo[client.Object, client.Object]().MapTo(d.currentGenerationDependents, d.stripObject)

	return d.marshalOrDie(&stripped)
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
			strconv.Itoa(int(d.agent.GetGeneration())): string(d.marshalDependents()),
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

func (d *dependentLifecycleManager) DeleteOrphanedDependents() result.Result[corev1.ConfigMap] {
	switch lifecycleCmFromCluster := d.getLifecycleCm(); lifecycleCmFromCluster.IsSuccess() {
	case true:

	default:
		return lifecycleCmFromCluster
	}
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
		jsonMarshaler: &jsonOrDie{
			OrDie: or_die.New[[]byte](),
		},
	}
}
