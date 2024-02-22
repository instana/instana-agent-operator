package status

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/semver/v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/collections/list"
	"github.com/instana/instana-agent-operator/pkg/env"
	instanaclient "github.com/instana/instana-agent-operator/pkg/k8s/client"
	"github.com/instana/instana-agent-operator/pkg/multierror"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/result"
)

// Conditions
const (
	ConditionTypeReconcileSucceeded = "ReconcileSucceeded"
	ConditionTypeAllAgentsReady     = "AllAgentsReady"
	CondtionTypeAllK8sSensorsReady  = "AllK8sSensorsReady"
)

func NewAgentStatusManager(k8sClient client.Client, eventRecorder record.EventRecorder) AgentStatusManager {
	return &agentStatusManager{
		k8sClient:       instanaclient.NewClient(k8sClient),
		eventRecorder:   eventRecorder,
		agentDaemonsets: make([]client.ObjectKey, 0, 1),
	}
}

type AgentStatusManager interface {
	AddAgentDaemonset(agentDaemonset client.ObjectKey)
	SetK8sSensorDeployment(k8sSensorDeployment client.ObjectKey)
	SetAgentConfigMap(agentConfigMap client.ObjectKey)
	UpdateAgentStatus(ctx context.Context, reconcileErr error) error
}

type agentStatusManager struct {
	k8sClient     instanaclient.InstanaAgentClient
	eventRecorder record.EventRecorder

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

func eventTypeFromCondition(condition metav1.Condition) string {
	switch condition.Status {
	case metav1.ConditionTrue:
		return corev1.EventTypeNormal
	default:
		return corev1.EventTypeWarning
	}
}

func (a *agentStatusManager) setConditionAndFireEvent(agentNew *instanav1.InstanaAgent, condition metav1.Condition) {
	meta.SetStatusCondition(&agentNew.Status.Conditions, condition)
	a.eventRecorder.Event(agentNew, eventTypeFromCondition(condition), condition.Reason, condition.Message)
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

func daemonsetIsAvailable(ds appsv1.DaemonSet) bool {
	switch status := ds.Status; {
	case ds.Generation != status.ObservedGeneration:
		return false
	case status.NumberMisscheduled != 0:
		return false
	case status.DesiredNumberScheduled != status.NumberAvailable:
		return false
	case status.DesiredNumberScheduled != status.UpdatedNumberScheduled:
		return false
	default:
		return true
	}
}

func (a *agentStatusManager) getAllAgentsReadyCondition(ctx context.Context) result.Result[metav1.Condition] {
	condition := metav1.Condition{
		Type:               ConditionTypeAllAgentsReady,
		Status:             "",
		ObservedGeneration: a.agentOld.GetGeneration(),
		Reason:             "",
		Message:            "",
	}

	dameonsets := make([]appsv1.DaemonSet, 0, len(a.agentDaemonsets))

	for _, key := range a.agentDaemonsets {
		var ds appsv1.DaemonSet
		switch res := a.k8sClient.GetAsResult(ctx, key, &ds); {
		case res.IsSuccess():
			dameonsets = append(dameonsets, ds)
		case res.IsFailure():
			_, err := res.Get()

			condition.Status = metav1.ConditionUnknown
			condition.Reason = "AgentDaemonsetInfoUnavailable"
			//goland:noinspection GoNilness
			msg := fmt.Sprintf(
				"failed to retrieve status of Agent Daemonset: %s due to error: %s",
				key.Name,
				err.Error(),
			)
			truncatedMsg := truncateMessage(msg)
			condition.Message = truncatedMsg

			return result.Of(condition, err)
		}
	}

	// TODO: Implement a robust readiness endpoint in the agent for the readiness probe?
	switch list.NewConditions(dameonsets).All(daemonsetIsAvailable) {
	case true:
		condition.Status = metav1.ConditionTrue
		condition.Reason = "AllDesiredAgentsRunning"
		condition.Message = "All desired Instana Agents are running and using up-to-date configuration"
	default:
		condition.Status = metav1.ConditionFalse
		condition.Reason = "NotAllDesiredAgentsRunning"
		condition.Message = "Not all desired Instana agents are running or some Agents are not using up-to-date configuration"
	}

	return result.OfSuccess(condition)
}

func (a *agentStatusManager) agentWithUpdatedStatus(
	ctx context.Context,
	reconcileErr error,
) result.Result[*instanav1.InstanaAgent] {
	errBuilder := multierror.NewMultiErrorBuilder()

	agentNew := a.agentOld.DeepCopy()
	logger := log.FromContext(ctx).WithName("agent-status-manager")

	// Handle Deprecated Status Fields

	agentNew.Status.Status = getAgentPhase(reconcileErr)
	agentNew.Status.Reason = getReason(reconcileErr)
	agentNew.Status.LastUpdate = metav1.Time{Time: time.Now()}

	a.getDaemonSet(ctx).
		OnSuccess(
			func(ds instanav1.ResourceInfo) {
				agentNew.Status.DaemonSet = ds
			},
		).
		OnFailure(errBuilder.AddSingle)

	a.getConfigMap(ctx).
		OnSuccess(
			func(cm instanav1.ResourceInfo) {
				agentNew.Status.ConfigMap = cm
			},
		).
		OnFailure(errBuilder.AddSingle)

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
	a.setConditionAndFireEvent(agentNew, a.getReconcileSucceededCondition(reconcileErr))
	allAgentsReadyCondition, _ := a.getAllAgentsReadyCondition(ctx).OnFailure(errBuilder.AddSingle).Get()
	a.setConditionAndFireEvent(agentNew, allAgentsReadyCondition)

	return result.Of(agentNew, errBuilder.Build())
}

func (a *agentStatusManager) UpdateAgentStatus(ctx context.Context, reconcileErr error) error {
	errBuilder := multierror.NewMultiErrorBuilder()

	agentNew, _ := a.agentWithUpdatedStatus(ctx, reconcileErr).OnFailure(errBuilder.AddSingle).Get()

	if err := a.k8sClient.Status().Patch(
		ctx,
		agentNew,
		client.MergeFrom(a.agentOld),
		client.FieldOwner(instanaclient.FieldOwnerName),
	); err != nil {
		errBuilder.AddSingle(err)
	}

	return errBuilder.Build()
}