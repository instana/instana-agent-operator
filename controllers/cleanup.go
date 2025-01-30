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
	"errors"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/e2e-framework/support/utils"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/controllers/reconciliation/helm"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/operator_utils"
)

func (r *InstanaAgentReconciler) cleanupHelmChart(
	ctx context.Context,
	agentOld *instanav1.InstanaAgent,
) reconcileReturn {
	agentNew := agentOld.DeepCopy()
	log := r.loggerFor(ctx, agentNew)

	if !controllerutil.RemoveFinalizer(agentNew, finalizerV1) {
		log.V(2).Info("no deprecated Helm chart installation detected based on finalizers present on agent CR")
		return reconcileContinue()
	} else if err := helm.NewHelmReconciliation(r.scheme, log).Delete(); !errors.Is(err, nil) {
		log.Error(err, "failed to uninstall deprecated Helm installation of Instana agent")
		return reconcileFailure(err)
	} else {
		log.V(1).Info("successfully uninstalled deprecated Helm installation of Instana agent")
		return r.updateAgent(ctx, agentOld, agentNew)
	}
}

func (r *InstanaAgentReconciler) cleanupDependents(
	ctx context.Context,
	agentOld *instanav1.InstanaAgent,
	operatorUtils operator_utils.OperatorUtils,
) reconcileReturn {
	agentNew := agentOld.DeepCopy()
	log := r.loggerFor(ctx, agentNew)

	if !controllerutil.RemoveFinalizer(agentNew, finalizerV3) {
		log.V(2).Info("agent finalizer not present, so no further cleanup is needed")
		return reconcileContinue()
	} else if err := operatorUtils.DeleteAll(); err != nil {
		log.Error(err, "failed to cleanup agent dependents during uninstall")
		return reconcileFailure(err)
	} else {
		log.V(1).Info("successfully cleaned up agent dependents during uninstall")
		return r.updateAgent(ctx, agentOld, agentNew)
	}
}

func (r *InstanaAgentReconciler) handleDeletion(
	ctx context.Context,
	agent *instanav1.InstanaAgent,
	operatorUtils operator_utils.OperatorUtils,
) reconcileReturn {
	log := r.loggerFor(ctx, agent)
	r.cleanupNodeLabels(ctx, agent)

	if agent.DeletionTimestamp == nil {
		log.V(2).Info("agent is not under deletion")
		return reconcileContinue()
	} else if cleanupChartRes := r.cleanupHelmChart(ctx, agent); cleanupChartRes.suppliesReconcileResult() {
		return cleanupChartRes
	} else if cleanupDependentsRes := r.cleanupDependents(
		ctx,
		agent,
		operatorUtils,
	); cleanupDependentsRes.suppliesReconcileResult() {
		return cleanupDependentsRes
	} else {
		return reconcileSuccess(ctrl.Result{})
	}
}

func (r *InstanaAgentReconciler) cleanupNodeLabels(
	ctx context.Context,
	agent *instanav1.InstanaAgent,
) {
	log := r.loggerFor(ctx, agent)
	p := utils.RunCommand("kubectl label node --all pool-")
	if p.Err() != nil {
		log.V(2).Info("Could not remove the labels from the nodes for multizone testing")
	}
}
