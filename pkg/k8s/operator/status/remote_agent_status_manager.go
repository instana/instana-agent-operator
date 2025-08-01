/*
(c) Copyright IBM Corp. 2025

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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/Masterminds/semver/v3"
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

type InstanaAgentRemoteStatusManager interface {
	AddAgentDeployment(agentDeployment client.ObjectKey) // Added method for Deployment
	SetAgentOld(agent *instanav1.InstanaAgentRemote)
	SetAgentSecretConfig(agentSecretConfig client.ObjectKey)
	UpdateAgentStatus(ctx context.Context, reconcileErr error) error
}

type instanaAgentRemoteStatusManager struct {
	instAgentClient   instanaclient.InstanaAgentClient
	eventRecorder     record.EventRecorder
	agentOld          *instanav1.InstanaAgentRemote
	agentDeployments  []client.ObjectKey // New field to store deployments
	agentSecretConfig client.ObjectKey
}

func NewInstanaAgentRemoteStatusManager(instAgentClient instanaclient.InstanaAgentClient, eventRecorder record.EventRecorder) InstanaAgentRemoteStatusManager {
	return &instanaAgentRemoteStatusManager{
		instAgentClient:  instAgentClient,
		eventRecorder:    eventRecorder,
		agentDeployments: make([]client.ObjectKey, 0, 1), // Initialize the deployments list
	}
}

// AddAgentDeployment adds a deployment to the list of agent deployments
func (a *instanaAgentRemoteStatusManager) AddAgentDeployment(agentDeployment client.ObjectKey) {
	if !list.NewContainsElementChecker(a.agentDeployments).Contains(agentDeployment) {
		a.agentDeployments = append(a.agentDeployments, agentDeployment)
	}
}

func (a *instanaAgentRemoteStatusManager) SetAgentOld(agent *instanav1.InstanaAgentRemote) {
	a.agentOld = agent.DeepCopy()
}

func (a *instanaAgentRemoteStatusManager) SetAgentSecretConfig(agentSecretConfig types.NamespacedName) {
	a.agentSecretConfig = agentSecretConfig
}

func (a *instanaAgentRemoteStatusManager) UpdateAgentStatus(ctx context.Context, reconcileErr error) (finalErr error) {
	defer recovery.Catch(&finalErr)

	if a.agentOld == nil {
		return nil
	}

	errBuilder := multierror.NewMultiErrorBuilder()

	agentNew, _ :=
		a.InstanaAgentRemoteWithUpdatedStatus(ctx, reconcileErr).
			OnFailure(errBuilder.AddSingle).
			Get()

	if err := a.instAgentClient.Status().Patch(
		ctx,
		agentNew,
		client.MergeFrom(a.agentOld),
		client.FieldOwner(instanaclient.FieldOwnerName),
	); err != nil {
		errBuilder.AddSingle(err)
	}

	return errBuilder.Build()
}

func (a *instanaAgentRemoteStatusManager) getDeployment(ctx context.Context) result.Result[instanav1.ResourceInfo] {
	if len(a.agentDeployments) != 1 {
		return result.OfSuccess(instanav1.ResourceInfo{})
	}

	deployment := a.instAgentClient.GetAsResult(ctx, a.agentDeployments[0], &appsv1.Deployment{})

	return result.Map(deployment, toResourceInfo)
}

func (a *instanaAgentRemoteStatusManager) getConfigSecret(ctx context.Context) result.Result[instanav1.ResourceInfo] {
	cm := a.instAgentClient.GetAsResult(ctx, a.agentSecretConfig, &corev1.Secret{})

	return result.Map(cm, toResourceInfo)
}

func (a *instanaAgentRemoteStatusManager) setConditionAndFireEvent(agentNew *instanav1.InstanaAgentRemote, condition metav1.Condition) {
	meta.SetStatusCondition(&agentNew.Status.Conditions, condition)
	a.eventRecorder.Event(agentNew, eventTypeFromCondition(condition), condition.Reason, condition.Message)
}

func (a *instanaAgentRemoteStatusManager) getReconcileSucceededCondition(reconcileErr error) metav1.Condition {
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
		res.Message = "most recent reconcile of instana agent remote CR completed without issue"
	default:
		res.Status = metav1.ConditionFalse
		res.Reason = "ReconcileFailed"
		// TODO: Error wrapping where propagating to add relevant info
		res.Message = truncateMessage(reconcileErr.Error())
	}

	return res
}

func (a *instanaAgentRemoteStatusManager) getAllAgentsAvailableCondition(ctx context.Context) result.Result[metav1.Condition] {
	condition := metav1.Condition{
		Type:               ConditionTypeAllAgentsAvailable,
		Status:             "",
		ObservedGeneration: a.agentOld.GetGeneration(),
		Reason:             "",
		Message:            "",
	}

	deployments := make([]appsv1.Deployment, 0, len(a.agentDeployments))

	for _, key := range a.agentDeployments {
		var deploy appsv1.Deployment
		switch res := a.instAgentClient.GetAsResult(ctx, key, &deploy); {
		case res.IsSuccess():
			deployments = append(deployments, deploy)
		case res.IsFailure():
			_, err := res.Get()

			condition.Status = metav1.ConditionUnknown
			condition.Reason = "AgentDeploymentInfoUnavailable"
			msg := fmt.Sprintf(
				"failed to retrieve status of Instana Agent Remote Deployment: %s due to error: %s",
				key.Name,
				err.Error(),
			)
			truncatedMsg := truncateMessage(msg)
			condition.Message = truncatedMsg

			return result.Of(condition, err)
		}
	}

	// Evaluate deployment availability (based on status conditions)
	if list.NewConditions(deployments).All(deploymentIsAvailableAndComplete) && len(deployments) > 0 {
		condition.Status = metav1.ConditionTrue
		condition.Reason = "AllDesiredAgentsAvailable"
		condition.Message = "All desired Instana Agent Remote are available and using up-to-date configuration"
	} else {
		condition.Status = metav1.ConditionFalse
		condition.Reason = "NotAllDesiredAgentsAvailable"
		condition.Message = "Not all desired Instana Agent Remote are available or some Agents are not using up-to-date configuration"
	}

	return result.OfSuccess(condition)
}

func (a *instanaAgentRemoteStatusManager) InstanaAgentRemoteWithUpdatedStatus(
	ctx context.Context,
	reconcileErr error,
) result.Result[*instanav1.InstanaAgentRemote] {
	errBuilder := multierror.NewMultiErrorBuilder()

	agentNew := a.agentOld.DeepCopy()
	logger := log.FromContext(ctx).WithName("instana-agent-remote-status-manager")

	a.getConfigSecret(ctx).
		OnSuccess(setStatusDotConfigSecretRemote(agentNew)).
		OnFailure(errBuilder.AddSingle)

	a.getDeployment(ctx).
		OnSuccess(setStatusDotDeployment(agentNew)). // New handler for Deployment status
		OnFailure(errBuilder.AddSingle)

	// Handle Conditions

	agentNew.Status.ObservedGeneration = pointer.To(a.agentOld.GetGeneration())

	result.Of(semver.NewVersion(env.GetOperatorVersion())).
		OnSuccess(setStatusDotOperatorVersionRemote(agentNew)).
		OnFailure(logOperatorVersionParseFailure(logger))

	agentNew.Status.Conditions = optional.Of(agentNew.Status.Conditions).GetOrDefault(make([]metav1.Condition, 0, 3))

	reconcileSucceededCondition := a.getReconcileSucceededCondition(reconcileErr)
	a.setConditionAndFireEvent(agentNew, reconcileSucceededCondition)

	allAgentsAvailableCondition, _ :=
		a.getAllAgentsAvailableCondition(ctx).
			OnFailure(errBuilder.AddSingle).
			Get()
	a.setConditionAndFireEvent(agentNew, allAgentsAvailableCondition)

	return result.Of(agentNew, errBuilder.Build())
}
