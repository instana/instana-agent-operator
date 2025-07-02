/*
(c) Copyright IBM Corp. 2024, 2025
*/

package status

import (
	"github.com/Masterminds/semver/v3"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/env"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/result"
)

// Conditions
const (
	ConditionTypeReconcileSucceeded    = "ReconcileSucceeded"
	ConditionTypeAllAgentsAvailable    = "AllAgentsAvailable"
	CondtionTypeAllK8sSensorsAvailable = "AllK8sSensorsAvailable"
)

func getAgentPhase(reconcileErr error) instanav1.AgentOperatorState {
	if reconcileErr != nil {
		return instanav1.OperatorStateFailed
	}
	return instanav1.OperatorStateRunning
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

func truncateMessage(message string) string {
	const limit = 32768

	if len(message) <= limit {
		return message
	} else {
		return message[:limit]
	}
}

func eventTypeFromCondition(condition metav1.Condition) string {
	if condition.Status == metav1.ConditionTrue {
		return corev1.EventTypeNormal
	}
	return corev1.EventTypeWarning
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

func setStatusDotDaemonset(agentNew *instanav1.InstanaAgent) func(ds instanav1.ResourceInfo) {
	return func(ds instanav1.ResourceInfo) {
		agentNew.Status.DaemonSet = ds
	}
}

func setStatusDotDeployment(agentNew *instanav1.RemoteAgent) func(deployment instanav1.ResourceInfo) {
	return func(deployment instanav1.ResourceInfo) {
		// Set the status of the agent based on the Deployment status
		agentNew.Status.Deployment = deployment
	}
}

func setStatusDotConfigSecret(agentNew *instanav1.InstanaAgent) func(cm instanav1.ResourceInfo) {
	return func(cm instanav1.ResourceInfo) {
		agentNew.Status.ConfigSecret = cm
	}
}

func setStatusDotNamespacesConfigmap(agentNew *instanav1.InstanaAgent) func(cm instanav1.ResourceInfo) {
	return func(cm instanav1.ResourceInfo) {
		agentNew.Status.NamespacesConfigMap = cm
	}
}

func setStatusDotConfigSecretRemote(agentNew *instanav1.RemoteAgent) func(cm instanav1.ResourceInfo) {
	return func(cm instanav1.ResourceInfo) {
		agentNew.Status.ConfigSecret = cm
	}
}

func setStatusDotOperatorVersion(agentNew *instanav1.InstanaAgent) func(version *semver.Version) {
	return func(version *semver.Version) {
		agentNew.Status.OperatorVersion = &instanav1.SemanticVersion{Version: *version}
	}
}

func setStatusDotOperatorVersionRemote(agentNew *instanav1.RemoteAgent) func(version *semver.Version) {
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
