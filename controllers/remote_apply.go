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

package controllers

import (
	"context"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	backends "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/backends"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	remoteagentdeployment "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/remote-agent/deployment"
	agentsecrets "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/remote-agent/secrets"
	keyssecret "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/remote-agent/secrets/keys-secret"
	tlssecret "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/remote-agent/secrets/tls-secret"
	agentserviceaccount "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/remote-agent/serviceaccount"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/operator_utils"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/status"
	corev1 "k8s.io/api/core/v1"
)

func getRemoteAgentDeployments(
	agent *instanav1.RemoteAgent,
	statusManager status.RemoteAgentStatusManager,
	additionalBackends []backends.K8SensorBackend,
	keysSecret *corev1.Secret,
) []builder.ObjectBuilder {
	builders := make([]builder.ObjectBuilder, 0, len(additionalBackends))

	for _, backend := range additionalBackends {
		builders = append(
			builders,
			remoteagentdeployment.NewDeploymentBuilder(agent, statusManager, backend, keysSecret),
		)
	}

	return builders
}

func (r *RemoteAgentReconciler) applyResources(
	ctx context.Context,
	agent *instanav1.RemoteAgent,
	operatorUtils operator_utils.RemoteOperatorUtils,
	statusManager status.RemoteAgentStatusManager,
	keysSecret *corev1.Secret,
	additionalBackends []backends.K8SensorBackend,
) reconcileReturn {
	log := r.loggerFor(ctx, agent)
	log.V(1).Info("applying Kubernetes resources for remote agent")

	builders := append(
		getRemoteAgentDeployments(agent, statusManager, additionalBackends, keysSecret),
		agentsecrets.NewConfigBuilder(agent, statusManager, keysSecret, additionalBackends),
		agentsecrets.NewContainerBuilder(agent, keysSecret),
		tlssecret.NewSecretBuilder(agent),
		agentserviceaccount.NewServiceAccountBuilder(agent),
		keyssecret.NewSecretBuilder(agent, additionalBackends),
	)
	if err := operatorUtils.ApplyAll(builders...); err != nil {
		log.Error(err, "failed to apply kubernetes resources for remote agent")
		return reconcileFailure(err)
	}

	log.V(1).Info("successfully applied kubernetes resources for remote agent")
	return reconcileContinue()
}
