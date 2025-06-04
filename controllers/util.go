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

package controllers

import (
	"context"
	"strconv"

	"github.com/go-logr/logr"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	backends "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/backends"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/operator_utils"
)

func (r *InstanaAgentReconciler) isOpenShift(ctx context.Context, operatorUtils operator_utils.OperatorUtils) (
	bool,
	reconcileReturn,
) {
	log := logf.FromContext(ctx)

	isOpenShiftRes, err := operatorUtils.ClusterIsOpenShift()
	if err != nil {
		log.Error(err, "failed to determine if cluster is OpenShift")
		return false, reconcileFailure(err)
	}
	log.V(1).Info("successfully detected whether cluster is OpenShift", "IsOpenShift", isOpenShiftRes)
	return isOpenShiftRes, reconcileContinue()
}

func (r *InstanaAgentReconciler) getK8SensorBackends(agent *instanav1.InstanaAgent) []backends.K8SensorBackend {
	k8SensorBackends := make([]backends.K8SensorBackend, 0, len(agent.Spec.Agent.AdditionalBackends)+1)
	k8SensorBackends = append(
		k8SensorBackends,
		*backends.NewK8SensorBackend("", agent.Spec.Agent.Key, agent.Spec.Agent.DownloadKey, agent.Spec.Agent.EndpointHost, agent.Spec.Agent.EndpointPort),
	)

	if len(agent.Spec.Agent.AdditionalBackends) == 0 {
		return k8SensorBackends
	}

	for i, additionalBackend := range agent.Spec.Agent.AdditionalBackends {
		k8SensorBackends = append(
			k8SensorBackends,
			*backends.NewK8SensorBackend("-"+strconv.Itoa(i+1), additionalBackend.Key, "", additionalBackend.EndpointHost, additionalBackend.EndpointPort),
		)
	}
	return k8SensorBackends
}

func (r *RemoteAgentReconciler) getK8SensorBackends(agent *instanav1.RemoteAgent) []backends.K8SensorBackend {
	k8SensorBackends := make([]backends.K8SensorBackend, 0, len(agent.Spec.Agent.AdditionalBackends)+1)
	k8SensorBackends = append(
		k8SensorBackends,
		*backends.NewK8SensorBackend("", agent.Spec.Agent.Key, agent.Spec.Agent.DownloadKey, agent.Spec.Agent.EndpointHost, agent.Spec.Agent.EndpointPort),
	)

	if len(agent.Spec.Agent.AdditionalBackends) == 0 {
		return k8SensorBackends
	}

	for i, additionalBackend := range agent.Spec.Agent.AdditionalBackends {
		k8SensorBackends = append(
			k8SensorBackends,
			*backends.NewK8SensorBackend("-"+strconv.Itoa(i+1), additionalBackend.Key, "", additionalBackend.EndpointHost, additionalBackend.EndpointPort),
		)
	}
	return k8SensorBackends
}

func (r *InstanaAgentReconciler) loggerFor(ctx context.Context, agent *instanav1.InstanaAgent) logr.Logger {
	return logf.FromContext(ctx).WithValues(
		"Generation",
		agent.Generation,
		"UID",
		agent.UID,
	)
}

func (r *RemoteAgentReconciler) loggerFor(ctx context.Context, agent *instanav1.RemoteAgent) logr.Logger {
	return logf.FromContext(ctx).WithValues(
		"Generation",
		agent.Generation,
		"UID",
		agent.UID,
	)
}
