/*
(c) Copyright IBM Corp. 2024
(c) Copyright Instana Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package status

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/collections/list"
	"github.com/instana/instana-agent-operator/pkg/env"
	instanaclient "github.com/instana/instana-agent-operator/pkg/k8s/client"
	"github.com/instana/instana-agent-operator/pkg/multierror"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
	"github.com/instana/instana-agent-operator/pkg/recovery"
	"github.com/instana/instana-agent-operator/pkg/result"
)

// Conditions
const (
	ConditionTypeReconcileSucceeded    = "ReconcileSucceeded"
	ConditionTypeAllAgentsAvailable    = "AllAgentsAvailable"
	CondtionTypeAllK8sSensorsAvailable = "AllK8sSensorsAvailable"
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
	SetAgentOld(agent *instanav1.InstanaAgent)
	SetK8sSensorDeployment(k8sSensorDeployment client.ObjectKey)
	SetAgentConfigMap(agentConfigMap client.ObjectKey)
	SetAgentConfigSecret(agentConfigSecret client.ObjectKey)
	UpdateAgentStatus(ctx context.Context, reconcileErr error) error
}

type agentStatusManager struct {
	k8sClient     instanaclient.InstanaAgentClient
	eventRecorder record.EventRecorder

	agentOld *instanav1.InstanaAgent

	agentDaemonsets     []client.ObjectKey
	k8sSensorDeployment client.ObjectKey
	agentConfigMap      client.ObjectKey
	agentConfigSecret   client.ObjectKey
}

func (a *agentStatusManager) SetAgentOld(agent *instanav1.InstanaAgent) {
	a.agentOld = agent.DeepCopy()
}

func (a *agentStatusManager) AddAgentDaemonset(agentDaemonset client.ObjectKey) {
	if !list.NewContainsElementChecker(a.agentDaemonsets).Contains(agentDaemonset) {
		a.agentDaemonsets = append(a.agentDaemonsets, agentDaemonset)
	}
}

func (a *agentStatusManager) SetK8sSensorDeployment(k8sSensorDeployment client.ObjectKey) {
	a.k8sSensorDeployment = k8sSensorDeployment
}

func (a *agentStatusManager) SetAgentConfigMap(agentConfigMap client.ObjectKey) {
	a.agentConfigMap = agentConfigMap
}

func (a *agentStatusManager) SetAgentConfigSecret(agentConfigSecret types.NamespacedName) {
	a.agentConfigSecret = agentConfigSecret
}

func getAgentPhase(reconcileErr error) instanav1.AgentOperatorState {
	switch reconcileErr {
	case nil:
		return instanav1.OperatorStateRunning
	default:
		return instanav1.OperatorStateFailed
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

func (a *agentStatusManager) getConfigSecret(ctx context.Context) result.Result[instanav1.ResourceInfo] {
	cm := a.k8sClient.GetAsResult(ctx, a.agentConfigSecret, &corev1.Secret{})

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
	//nolint:exhaustive
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
	case optional.Of(status).IsNotPresent():
		return false
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

func (a *agentStatusManager) getAllAgentsAvailableCondition(ctx context.Context) result.Result[metav1.Condition] {
	condition := metav1.Condition{
		Type:               ConditionTypeAllAgentsAvailable,
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
		condition.Reason = "AllDesiredAgentsAvailable"
		condition.Message = "All desired Instana Agents are available and using up-to-date configuration"
	default:
		condition.Status = metav1.ConditionFalse
		condition.Reason = "NotAllDesiredAgentsAvailable"
		condition.Message = "Not all desired Instana agents are available or some Agents are not using up-to-date configuration"
	}

	return result.OfSuccess(condition)
}

type deploymentConditionsMap map[appsv1.DeploymentConditionType]appsv1.DeploymentCondition

func deploymentConditionsAsMap(conditions []appsv1.DeploymentCondition) deploymentConditionsMap {
	res := make(deploymentConditionsMap, len(conditions))

	for _, condition := range conditions {
		res[condition.Type] = condition
	}

	return res
}

func deploymentHasMinimumAvailability(conditions deploymentConditionsMap) bool {
	switch condition, isPresent := conditions[appsv1.DeploymentAvailable]; isPresent {
	case true:
		return condition.Status == corev1.ConditionTrue
	default:
		return false
	}
}

func deploymentIsComplete(conditions deploymentConditionsMap) bool {
	switch condition, isPresent := conditions[appsv1.DeploymentProgressing]; isPresent {
	case true:
		return condition.Status == corev1.ConditionTrue && condition.Reason == "NewReplicaSetAvailable"
	default:
		return false
	}
}

func deploymentHasReplicaFailures(conditions deploymentConditionsMap) bool {
	switch condition, isPresent := conditions[appsv1.DeploymentReplicaFailure]; isPresent {
	case true:
		return condition.Status == corev1.ConditionTrue
	default:
		return false
	}
}

func deploymentIsAvailableAndComplete(dpl appsv1.Deployment) bool {
	conditions := deploymentConditionsAsMap(dpl.Status.Conditions)

	switch {
	case dpl.Status.ObservedGeneration != dpl.Generation:
		return false
	case !deploymentHasMinimumAvailability(conditions):
		return false
	case !deploymentIsComplete(conditions):
		return false
	case deploymentHasReplicaFailures(conditions):
		return false
	default:
		return true
	}
}

func (a *agentStatusManager) updateWasPerformed() bool {
	switch operatorVersion, _ := semver.NewVersion(env.GetOperatorVersion()); {
	case a.agentOld.Status.ObservedGeneration == nil:
		return false
	case *a.agentOld.Status.ObservedGeneration != a.agentOld.Generation:
		return true
	case a.agentOld.Status.OperatorVersion == nil:
		return false
	case operatorVersion == nil:
		return true
	case !a.agentOld.Status.OperatorVersion.Version.Equal(operatorVersion):
		return true
	default:
		return false
	}
}

func (a *agentStatusManager) getAllK8sSensorsAvailableCondition(ctx context.Context) result.Result[metav1.Condition] {
	condition := metav1.Condition{
		Type:               CondtionTypeAllK8sSensorsAvailable,
		Status:             "",
		ObservedGeneration: a.agentOld.GetGeneration(),
		Reason:             "",
		Message:            "",
	}

	var deployment appsv1.Deployment

	if res := a.k8sClient.GetAsResult(ctx, a.k8sSensorDeployment, &deployment); res.IsFailure() {
		_, err := res.Get()

		condition.Status = metav1.ConditionUnknown
		condition.Reason = "K8sSensorDeploymentInfoUnavailable"
		//goland:noinspection GoNilness
		msg := fmt.Sprintf(
			"failed to retrieve status of K8sSensor Deployment: %s due to error: %s",
			a.k8sSensorDeployment.Name,
			err.Error(),
		)
		truncatedMsg := truncateMessage(msg)
		condition.Message = truncatedMsg

		return result.Of(condition, err)
	}

	switch deploymentIsAvailableAndComplete(deployment) {
	case true:
		condition.Status = metav1.ConditionTrue
		condition.Reason = "AllDesiredK8sSensorsAvailable"
		condition.Message = "All desired K8sSensors are available and using up-to-date configuration"
	default:
		condition.Status = metav1.ConditionFalse
		condition.Reason = "NotAllDesiredK8sSensorsAvailable"
		condition.Message = "Not all desired K8sSensors are available or some K8sSensors are not using up-to-date configuration"
	}

	return result.OfSuccess(condition)
}

func setStatusDotDaemonset(agentNew *instanav1.InstanaAgent) func(ds instanav1.ResourceInfo) {
	return func(ds instanav1.ResourceInfo) {
		agentNew.Status.DaemonSet = ds
	}
}

func setStatusDotConfigMap(agentNew *instanav1.InstanaAgent) func(cm instanav1.ResourceInfo) {
	return func(cm instanav1.ResourceInfo) {
		agentNew.Status.ConfigMap = cm
	}
}

func setStatusDotConfigSecret(agentNew *instanav1.InstanaAgent) func(cm instanav1.ResourceInfo) {
	return func(cm instanav1.ResourceInfo) {
		agentNew.Status.ConfigSecret = cm
	}
}

func setStatusDotOperatorVersion(agentNew *instanav1.InstanaAgent) func(version *semver.Version) {
	return func(version *semver.Version) {
		agentNew.Status.OperatorVersion = &instanav1.SemanticVersion{Version: *version}
	}
}

func logOperatorVersionParseFailure(logger logr.Logger) func(err error) {
	return func(err error) {
		logger.Error(
			err,
			"operator version is not a valid semantic version",
			"OperatorVersion",
			env.GetOperatorVersion(),
		)
	}
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
		OnSuccess(setStatusDotDaemonset(agentNew)).
		OnFailure(errBuilder.AddSingle)

	a.getConfigMap(ctx).
		OnSuccess(setStatusDotConfigMap(agentNew)).
		OnFailure(errBuilder.AddSingle)

	a.getConfigSecret(ctx).
		OnSuccess(setStatusDotConfigSecret(agentNew)).
		OnFailure(errBuilder.AddSingle)

	if a.updateWasPerformed() {
		agentNew.Status.OldVersionsUpdated = true
	}

	// Handle New Status Fields

	agentNew.Status.ObservedGeneration = pointer.To(a.agentOld.GetGeneration())

	result.Of(semver.NewVersion(env.GetOperatorVersion())).
		OnSuccess(setStatusDotOperatorVersion(agentNew)).
		OnFailure(logOperatorVersionParseFailure(logger))

	// Handle Conditions
	agentNew.Status.Conditions = optional.Of(agentNew.Status.Conditions).GetOrDefault(make([]metav1.Condition, 0, 3))

	reconcileSucceededCondition := a.getReconcileSucceededCondition(reconcileErr)
	a.setConditionAndFireEvent(agentNew, reconcileSucceededCondition)

	allAgentsAvailableCondition, _ :=
		a.getAllAgentsAvailableCondition(ctx).
			OnFailure(errBuilder.AddSingle).
			Get()
	a.setConditionAndFireEvent(agentNew, allAgentsAvailableCondition)

	allK8sSensorsAvailableCondition, _ :=
		a.getAllK8sSensorsAvailableCondition(ctx).
			OnFailure(errBuilder.AddSingle).
			Get()
	a.setConditionAndFireEvent(agentNew, allK8sSensorsAvailableCondition)

	return result.Of(agentNew, errBuilder.Build())
}

func (a *agentStatusManager) UpdateAgentStatus(ctx context.Context, reconcileErr error) (finalErr error) {
	defer recovery.Catch(&finalErr)

	if a.agentOld == nil {
		return nil
	}

	errBuilder := multierror.NewMultiErrorBuilder()

	agentNew, _ :=
		a.agentWithUpdatedStatus(ctx, reconcileErr).
			OnFailure(errBuilder.AddSingle).
			Get()

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
