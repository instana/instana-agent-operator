package status

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
)

func NewAgentStatusManager(k8sClient client.Client) AgentStatusManager {
	return &agentStatusManager{
		k8sClient:       k8sClient,
		agentDaemonsets: make([]client.ObjectKey, 0, 1),
	}
}

type AgentStatusManager interface {
	AddAgentDaemonset(agentDaemonset client.ObjectKey)
	SetK8sSensorDeployment(k8sSensorDeployment client.ObjectKey)
	SetAgentConfigMap(agentConfigMap client.ObjectKey)
}

type agentStatusManager struct {
	k8sClient client.Client

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

func (a *agentStatusManager) agentWithUpdatedStatus(
	ctx context.Context,
	reconcileErr error,
	agentOld *instanav1.InstanaAgent,
) *instanav1.InstanaAgent {
	agentNew := agentOld.DeepCopy()

	agentNew.Status.Status = getAgentPhase(reconcileErr)
	agentNew.Status.Reason = getReason(reconcileErr)
	agentNew.Status.LastUpdate = metav1.Time{Time: time.Now()}

	panic("TODO")
}
