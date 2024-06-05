package controllers

import (
	"context"

	"github.com/go-logr/logr"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/operator_utils"
)

func (r *InstanaAgentReconciler) isOpenShift(ctx context.Context, operatorUtils operator_utils.OperatorUtils) (
	bool,
	reconcileReturn,
) {
	log := logf.FromContext(ctx)

	isOpenShiftRes := operatorUtils.ClusterIsOpenShift()
	answer, err := isOpenShiftRes.Get()

	switch isOpenShiftRes.IsSuccess() {
	case true:
		log.V(1).Info("successfully detected whether cluster is OpenShift", "IsOpenShift", answer)
		return answer, reconcileContinue()
	default:
		log.Error(err, "failed to determine if cluster is OpenShift")
		return false, reconcileFailure(err)
	}
}

func (r *InstanaAgentReconciler) loggerFor(ctx context.Context, agent *instanav1.InstanaAgent) logr.Logger {
	return logf.FromContext(ctx).WithValues(
		"Generation",
		agent.Generation,
		"UID",
		agent.UID,
	)
}
