package status

import (
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/result"

	instanaclient "github.com/instana/instana-agent-operator/pkg/k8s/client"
)

func NewAgentStatusManager(k8sClient instanaclient.InstanaAgentClient) AgentStatusManager {
	return &agentStatusManager{
		k8sClient:       k8sClient,
		agentDaemonsets: make([]client.ObjectKey, 0, 1),
	}
}

type AgentStatusManager interface {
	AddAgentDaemonset(agentDaemonset client.ObjectKey)
	SetK8sSensorDeployment(k8sSensorDeployment client.ObjectKey)
	SetAgentConfigMap(agentConfigMap client.ObjectKey)
	UpdateAgentStatus(ctx context.Context, reconcileErr error, agent client.ObjectKey) error
}

type agentStatusManager struct {
	k8sClient instanaclient.InstanaAgentClient

	agentDaemonsets     []client.ObjectKey
	k8sSensorDeployment client.ObjectKey
	agentConfigMap      client.ObjectKey
}

func (a *agentStatusManager) AddAgentDaemonset(agentDaemonset client.ObjectKey) {
	a.agentDaemonsets = append(a.agentDaemonsets, agentDaemonset)
}

func (a *agentStatusManager) SetK8sSensorDeployment(k8sSensorDeployment client.ObjectKey) {
	a.k8sSensorDeployment = k8sSensorDeployment
}

func (a *agentStatusManager) SetAgentConfigMap(agentConfigMap client.ObjectKey) {
	a.agentConfigMap = agentConfigMap
}

func getAgentPhase(reconcileErr error) instanav1.AgentOperatorState {
	switch reconcileErr {
	case nil:
		return instanav1.OperatorStateFailed
	default:
		return instanav1.OperatorStateRunning
	}
}

func getReason(reconcileErr error) string {
	switch reconcileErr {
	case nil:
		return ""
	default:
		return reconcileErr.Error()
	}
}

func toResourceInfo(obj client.Object) result.Result[instanav1.ResourceInfo] {
	return result.OfSuccess(
		instanav1.ResourceInfo{
			Name: obj.GetName(),
			UID:  string(obj.GetUID()),
		},
	)
}

func (a *agentStatusManager) getDaemonSet(ctx context.Context) result.Result[instanav1.ResourceInfo] {
	if len(a.agentDaemonsets) != 1 {
		return result.OfSuccess(instanav1.ResourceInfo{})
	}

	ds := a.k8sClient.GetAsResult(ctx, a.agentDaemonsets[0], &appsv1.DaemonSet{})

	return result.Map(ds, toResourceInfo)
}

func (a *agentStatusManager) getConfigMap(ctx context.Context) result.Result[instanav1.ResourceInfo] {
	cm := a.k8sClient.GetAsResult(ctx, a.agentConfigMap, &corev1.ConfigMap{})

	return result.Map(cm, toResourceInfo)
}

func (a *agentStatusManager) agentWithUpdatedStatus(
	ctx context.Context,
	reconcileErr error,
	agentOld *instanav1.InstanaAgent,
) result.Result[*instanav1.InstanaAgent] {
	agentNew := agentOld.DeepCopy()

	agentNew.Status.Status = getAgentPhase(reconcileErr)
	agentNew.Status.Reason = getReason(reconcileErr)
	agentNew.Status.LastUpdate = metav1.Time{Time: time.Now()}

	switch res := a.getDaemonSet(ctx); {
	case res.IsSuccess():
		ds, _ := res.Get()
		agentNew.Status.DaemonSet = ds
	case res.IsFailure():
		_, err := res.Get()
		return result.OfFailure[*instanav1.InstanaAgent](err)
	}

	switch res := a.getConfigMap(ctx); {
	case res.IsSuccess():
		cm, _ := res.Get()
		agentNew.Status.ConfigMap = cm
	case res.IsFailure():
		_, err := res.Get()
		return result.OfFailure[*instanav1.InstanaAgent](err)
	}

	return result.OfSuccess(agentNew)
}

func (a *agentStatusManager) UpdateAgentStatus(ctx context.Context, reconcileErr error, agent client.ObjectKey) error {
	agentOld := &instanav1.InstanaAgent{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "instana.io/v1",
			Kind:       "InstanaAgent",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	}

	switch res := a.agentWithUpdatedStatus(ctx, reconcileErr, agentOld); {
	case res.IsSuccess():
		agentNew, _ := res.Get()
		return a.k8sClient.Status().Patch(
			ctx,
			agentNew,
			client.MergeFrom(agentOld),
			client.FieldOwner(instanaclient.FieldOwnerName),
		) // TODO: Observed Generation or use update?
	default:
		_, err := res.Get()
		return err
	}
}
