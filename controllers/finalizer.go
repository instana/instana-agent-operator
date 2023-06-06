package controllers

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	v1 "github.com/instana/instana-agent-operator/api/v1"
)

func (r *InstanaAgentReconciler) addOrUpdateFinalizers(ctx context.Context, agent *v1.InstanaAgent) reconcileReturn {
	switch removeHelmChartRes := r.cleanupHelmChart(ctx, agent); {
	case removeHelmChartRes.suppliesReconcileResult():
		return removeHelmChartRes
	case controllerutil.AddFinalizer(agent, finalizerV3):
		return r.updateAgent(ctx, agent)
	default:
		return reconcileContinue()
	}
}
