package controllers

import (
	"context"
	"errors"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	instanaclient "github.com/instana/instana-agent-operator/pkg/k8s/client"
)

func (r *InstanaAgentReconciler) getAgent(ctx context.Context, req ctrl.Request) (
	*instanav1.InstanaAgent,
	reconcileReturn,
) {
	var agent instanav1.InstanaAgent

	log := logf.FromContext(ctx)

	switch err := r.client.Get(ctx, req.NamespacedName, &agent); {
	case k8serrors.IsNotFound(err):
		log.V(10).Info("attempted to reconcile agent CR that could not be found")
		return nil, reconcileSuccess(ctrl.Result{})
	case !errors.Is(err, nil):
		log.Error(err, "failed to retrieve info about agent CR")
		return nil, reconcileFailure(err)
	default:
		log.V(1).Info("successfully retrieved agent CR info")
		return &agent, reconcileContinue()
	}
}

func (r *InstanaAgentReconciler) updateAgent(
	ctx context.Context,
	agentOld *instanav1.InstanaAgent,
	agentNew *instanav1.InstanaAgent,
) reconcileReturn {
	log := r.loggerFor(ctx, agentNew)

	switch err := r.client.Patch(
		ctx,
		agentNew,
		client.MergeFrom(agentOld),
		client.FieldOwner(instanaclient.FieldOwnerName),
	); errors.Is(err, nil) {
	case true:
		log.V(1).Info("successfully applied updates to agent CR")
		return reconcileSuccess(ctrl.Result{Requeue: true})
	default:
		if !k8serrors.IsNotFound(err) {
			log.Error(err, "failed to apply updates to agent CR")
		}
		return reconcileFailure(err)
	}
}
