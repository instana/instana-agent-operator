package controllers

import (
	"context"
	"errors"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/operator_utils"
)

func (r *InstanaAgentReconciler) cleanupHelmChart(ctx context.Context, agent *instanav1.InstanaAgent) reconcileReturn {
	if !controllerutil.RemoveFinalizer(agent, finalizerV1) {
		return reconcileContinue()
	} else if err := r.chartRemover.Delete(); !errors.Is(err, nil) {
		return reconcileFailure(err)
	} else {
		return r.updateAgent(ctx, agent)
	}
}

func (r *InstanaAgentReconciler) cleanupDependents(
	ctx context.Context,
	agent *instanav1.InstanaAgent,
	operatorUtils operator_utils.OperatorUtils,
) reconcileReturn {
	if !controllerutil.RemoveFinalizer(agent, finalizerV3) {
		return reconcileContinue()
	} else if deleteRes := operatorUtils.DeleteAll(); deleteRes.IsFailure() {
		_, err := deleteRes.Get()
		return reconcileFailure(err)
	} else {
		return r.updateAgent(ctx, agent)
	}
}

func (r *InstanaAgentReconciler) handleDeletion(
	ctx context.Context,
	agent *instanav1.InstanaAgent,
	operatorUtils operator_utils.OperatorUtils,
) reconcileReturn {
	if agent.DeletionTimestamp == nil {
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
