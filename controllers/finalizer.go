package controllers

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	v1 "github.com/instana/instana-agent-operator/api/v1"
)

func (r *InstanaAgentReconciler) addOrUpdateFinalizers(ctx context.Context, agentOld *v1.InstanaAgent) reconcileReturn {
	agentNew := agentOld.DeepCopy()

	switch removeHelmChartRes := r.cleanupHelmChart(ctx, agentNew); {
	case removeHelmChartRes.suppliesReconcileResult():
		return removeHelmChartRes
	case controllerutil.AddFinalizer(agentNew, finalizerV3):
		r.loggerFor(ctx, agentNew).V(1).Info("adding agent finalizer to agent CR")
		return r.updateAgent(ctx, agentOld, agentNew)
	default:
		return reconcileContinue()
	}
}
