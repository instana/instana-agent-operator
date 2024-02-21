package status

import (
	"context"
	"time"

	"github.com/Masterminds/semver/v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/env"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/result"

	instanaclient "github.com/instana/instana-agent-operator/pkg/k8s/client"
)

// Conditions
const (
	ConditionTypeReconcileSucceeded = "ReconcileSucceeded"
	ConditionTypeAllAgentsReady     = "AllAgentsReady"
	CondtionTypeAllK8sSensorsReady  = "AllK8sSensorsReady"
)

func NewAgentStatusManager(k8sClient client.Client) AgentStatusManager {
	return &agentStatusManager{
		k8sClient:       instanaclient.NewClient(k8sClient),
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

	agentOld *instanav1.InstanaAgent

	agentDaemonsets     []client.ObjectKey
	k8sSensorDeployment client.ObjectKey
	agentConfigMap      client.ObjectKey
}

func (a *agentStatusManager) SetAgentOld(agent *instanav1.InstanaAgent) {
	a.agentOld = agent.DeepCopy()
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

func truncateMessage(message string) string {
	const limit = 32768

	if len(message) <= limit {
		return message
	} else {
		return message[:limit]
	}
}

func (a *agentStatusManager) getReconcileSucceededCondition(reconcileErr error) metav1.Condition {
	res := metav1.Condition{
		Type:               ConditionTypeReconcileSucceeded,
		Status:             "",
		ObservedGeneration: a.agentOld.GetGeneration(),
		Reason:             "",
		Message:            "",
	}

	switch reconcileErr {
	case nil:
		res.Status = metav1.ConditionTrue
		res.Reason = "ReconcileSucceeded"
		res.Message = "most recent reconcile of agent CR completed without issue"
	default:
		res.Status = metav1.ConditionFalse
		res.Reason = "ReconcileFailed"
		// TODO: Error wrapping where propagating to add relevant info
		res.Message = truncateMessage(reconcileErr.Error())
	}

	return res
}

func (a *agentStatusManager) agentWithUpdatedStatus(
	ctx context.Context,
	reconcileErr error,
) result.Result[*instanav1.InstanaAgent] {
	agentNew := a.agentOld.DeepCopy()
	logger := log.FromContext(ctx).WithName("agent-status-manager")

	// Handle Deprecated Status Fields

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

	// Handle New Status Fields

	agentNew.Status.ObservedGeneration = a.agentOld.GetGeneration()

	result.Of(semver.NewVersion(env.GetOperatorVersion())).
		OnSuccess(
			func(version *semver.Version) {
				agentNew.Status.OperatorVersion = instanav1.SemanticVersion{Version: *version}
			},
		).
		OnFailure(
			func(err error) {
				logger.Error(
					err,
					"operator version is not a valid semantic version",
					"OperatorVersion",
					env.GetOperatorVersion(),
				)
			},
		)

	// Handle Conditions

	agentNew.Status.Conditions = optional.Of(agentNew.Status.Conditions).GetOrDefault(make([]metav1.Condition, 0, 3))
	meta.SetStatusCondition(&agentNew.Status.Conditions, a.getReconcileSucceededCondition(reconcileErr))

	return result.OfSuccess(agentNew)
}

func (a *agentStatusManager) UpdateAgentStatus(ctx context.Context, reconcileErr error, agent client.ObjectKey) error {
	switch res := a.agentWithUpdatedStatus(ctx, reconcileErr); {
	case res.IsSuccess():
		agentNew, _ := res.Get()
		return a.k8sClient.Status().Patch(
			ctx,
			agentNew,
			client.MergeFrom(a.agentOld),
			client.FieldOwner(instanaclient.FieldOwnerName),
		)
	default:
		_, err := res.Get()
		return err
	}
}
