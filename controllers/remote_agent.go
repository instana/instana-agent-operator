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
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	instanaclient "github.com/instana/instana-agent-operator/pkg/k8s/client"
)

func (r *RemoteAgentReconciler) getRemoteAgent(ctx context.Context, req ctrl.Request) (
	*instanav1.RemoteAgent,
	reconcileReturn,
) {
	var agent instanav1.RemoteAgent

	log := logf.FromContext(ctx)

	switch err := r.client.Get(ctx, req.NamespacedName, &agent); {
	case k8serrors.IsNotFound(err):
		log.V(10).Info("attempted to reconcile remote agent CR that could not be found")
		return nil, reconcileSuccess(ctrl.Result{})
	case !errors.Is(err, nil):
		log.Error(err, "failed to retrieve info about remote agent CR")
		return nil, reconcileFailure(err)
	default:
		log.V(1).Info("successfully retrieved remote agent CR info")
		return &agent, reconcileContinue()
	}
}

func (r *RemoteAgentReconciler) getAgent(ctx context.Context, namespace, hostAgentName string) (*instanav1.InstanaAgent, reconcileReturn) {
	var agent instanav1.InstanaAgent
	err := r.client.Get(ctx, types.NamespacedName{Name: hostAgentName, Namespace: namespace}, &agent)
	if k8serrors.IsNotFound(err) {
		return nil, reconcileSuccess(ctrl.Result{})
	}
	if err != nil {
		return nil, reconcileFailure(err)
	}
	return &agent, reconcileContinue()
}

func (r *RemoteAgentReconciler) updateAgent(
	ctx context.Context,
	agentOld *instanav1.RemoteAgent,
	agentNew *instanav1.RemoteAgent,
) reconcileReturn {
	log := r.loggerFor(ctx, agentNew)

	switch err := r.client.Patch(
		ctx,
		agentNew,
		client.MergeFrom(agentOld),
		client.FieldOwner(instanaclient.FieldOwnerName),
	); errors.Is(err, nil) {
	case true:
		log.V(1).Info("successfully applied updates to remote agent CR")
		return reconcileSuccess(ctrl.Result{Requeue: true})
	default:
		if !k8serrors.IsNotFound(err) {
			log.Error(err, "failed to apply updates to remote agent CR")
		}
		return reconcileFailure(err)
	}
}
