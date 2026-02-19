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
	"fmt"
	"strconv"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	backends "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/backends"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/operator_utils"
)

func (r *InstanaAgentReconciler) isOpenShift(
	ctx context.Context,
	operatorUtils operator_utils.OperatorUtils,
) (
	bool,
	reconcileReturn,
) {
	log := logf.FromContext(ctx)

	isOpenShiftRes, err := operatorUtils.ClusterIsOpenShift()
	if err != nil {
		log.Error(err, "failed to determine if cluster is OpenShift")
		return false, reconcileFailure(err)
	}
	log.V(1).
		Info("successfully detected whether cluster is OpenShift", "IsOpenShift", isOpenShiftRes)
	return isOpenShiftRes, reconcileContinue()
}

// shouldSetPersistHostUniqueIDEnvVar determines whether to set the
// INSTANA_PERSIST_HOST_UNIQUE_ID environment variable.
// The logic is:
// - If DaemonSet doesn't exist: set it (new deployment)
// - If DaemonSet exists and already has the env var: keep it
// - If DaemonSet exists but doesn't have the env var: don't add it (upgrade scenario)
func (r *InstanaAgentReconciler) shouldSetPersistHostUniqueIDEnvVar(
	ctx context.Context,
	agent *instanav1.InstanaAgent,
	zone *instanav1.Zone,
) (bool, reconcileReturn) {
	log := logf.FromContext(ctx)

	// Determine the DaemonSet name based on whether we're using zones
	dsName := agent.Name
	if zone != nil {
		dsName = fmt.Sprintf("%s-%s", agent.Name, zone.Name.Name)
	}

	// Try to get the existing DaemonSet
	existingDS := &appsv1.DaemonSet{}
	err := r.client.Get(ctx, types.NamespacedName{
		Name:      dsName,
		Namespace: agent.Namespace,
	}, existingDS)

	// If DaemonSet doesn't exist, this is a new deployment - set the env var
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.V(1).
				Info("DaemonSet not found, will set INSTANA_PERSIST_HOST_UNIQUE_ID", "daemonset", dsName)
			return true, reconcileContinue()
		}
		// On other errors, abort reconcile to avoid removing an existing env var.
		log.Error(
			err,
			"failed to check existing DaemonSet",
			"daemonset",
			dsName,
		)
		return false, reconcileFailure(err)
	}

	// DaemonSet exists - check if it already has the env var
	for _, container := range existingDS.Spec.Template.Spec.Containers {
		if container.Name == "instana-agent" {
			for _, env := range container.Env {
				if env.Name == "INSTANA_PERSIST_HOST_UNIQUE_ID" {
					// Env var already exists, keep it
					log.V(1).
						Info("DaemonSet already has INSTANA_PERSIST_HOST_UNIQUE_ID, will keep it", "daemonset", dsName)
					return true, reconcileContinue()
				}
			}
			// Container found but env var doesn't exist - don't add it (upgrade scenario)
			log.V(1).
				Info(
					"DaemonSet exists without INSTANA_PERSIST_HOST_UNIQUE_ID, will not add it (upgrade)",
					"daemonset",
					dsName,
				)
			return false, reconcileContinue()
		}
	}

	// Container not found (shouldn't happen), default to not setting it
	log.V(1).
		Info(
			"instana-agent container not found in DaemonSet, will not set INSTANA_PERSIST_HOST_UNIQUE_ID",
			"daemonset",
			dsName,
		)
	return false, reconcileContinue()
}

func (r *InstanaAgentReconciler) getK8SensorBackends(
	agent *instanav1.InstanaAgent,
) []backends.K8SensorBackend {
	k8SensorBackends := make(
		[]backends.K8SensorBackend,
		0,
		len(agent.Spec.Agent.AdditionalBackends)+1,
	)
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

func (r *InstanaAgentRemoteReconciler) getRemoteSensorBackends(
	agent *instanav1.InstanaAgentRemote,
) []backends.RemoteSensorBackend {
	remoteSensorBackends := make(
		[]backends.RemoteSensorBackend,
		0,
		len(agent.Spec.Agent.AdditionalBackends)+1,
	)
	remoteSensorBackends = append(
		remoteSensorBackends,
		*backends.NewRemoteSensorBackend("", agent.Spec.Agent.Key, agent.Spec.Agent.DownloadKey, agent.Spec.Agent.EndpointHost, agent.Spec.Agent.EndpointPort),
	)

	if len(agent.Spec.Agent.AdditionalBackends) == 0 {
		return remoteSensorBackends
	}

	for i, additionalBackend := range agent.Spec.Agent.AdditionalBackends {
		remoteSensorBackends = append(
			remoteSensorBackends,
			*backends.NewRemoteSensorBackend("-"+strconv.Itoa(i+1), additionalBackend.Key, "", additionalBackend.EndpointHost, additionalBackend.EndpointPort),
		)
	}
	return remoteSensorBackends
}

func (r *InstanaAgentReconciler) loggerFor(
	ctx context.Context,
	agent *instanav1.InstanaAgent,
) logr.Logger {
	return logf.FromContext(ctx).WithValues(
		"Generation",
		agent.Generation,
		"UID",
		agent.UID,
	)
}

func (r *InstanaAgentRemoteReconciler) loggerFor(
	ctx context.Context,
	agent *instanav1.InstanaAgentRemote,
) logr.Logger {
	return logf.FromContext(ctx).WithValues(
		"Generation",
		agent.Generation,
		"UID",
		agent.UID,
	)
}
