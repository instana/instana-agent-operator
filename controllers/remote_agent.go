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
	"errors"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	instanaclient "github.com/instana/instana-agent-operator/pkg/k8s/client"
)

func (r *InstanaAgentRemoteReconciler) getInstanaAgentRemote(ctx context.Context, req ctrl.Request) (
	*instanav1.InstanaAgentRemote,
	reconcileReturn,
) {
	var agent instanav1.InstanaAgentRemote

	log := logf.FromContext(ctx)

	switch err := r.client.Get(ctx, req.NamespacedName, &agent); {
	case k8serrors.IsNotFound(err):
		log.V(10).Info("attempted to reconcile instana agent remote CR that could not be found")
		return nil, reconcileSuccess(ctrl.Result{})
	case !errors.Is(err, nil):
		log.Error(err, "failed to retrieve info about instana agent remote CR")
		return nil, reconcileFailure(err)
	default:
		log.V(1).Info("successfully retrieved instana agent remote CR info")
		return &agent, reconcileContinue()
	}
}

func (r *InstanaAgentRemoteReconciler) updateAgent(
	ctx context.Context,
	agentOld *instanav1.InstanaAgentRemote,
	agentNew *instanav1.InstanaAgentRemote,
) reconcileReturn {
	log := r.loggerFor(ctx, agentNew)

	switch err := r.client.Patch(
		ctx,
		agentNew,
		client.MergeFrom(agentOld),
		client.FieldOwner(instanaclient.FieldOwnerName),
	); errors.Is(err, nil) {
	case true:
		log.V(1).Info("successfully applied updates to instana agent remote CR")
		return reconcileSuccess(ctrl.Result{Requeue: true})
	default:
		if !k8serrors.IsNotFound(err) {
			log.Error(err, "failed to apply updates to instana agent remote CR")
		}
		return reconcileFailure(err)
	}
}
