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

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	instanaclient "github.com/instana/instana-agent-operator/pkg/k8s/client"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/lifecycle"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/operator_utils"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/status"
	"github.com/instana/instana-agent-operator/pkg/multierror"
	"github.com/instana/instana-agent-operator/pkg/recovery"
)

// Add will create a new Instana Agent Remote Controller and add this to the Manager for reconciling
func AddRemote(mgr manager.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&instanav1.InstanaAgentRemote{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&corev1.Service{}).
		WithEventFilter(filterPredicateRemote()).
		Complete(
			NewInstanaAgentRemoteReconciler(
				mgr.GetClient(),
				mgr.GetScheme(),
				mgr.GetEventRecorderFor("instana-agent-remote-controller"),
			),
		)
}

// NewInstanaAgentReconciler initializes a new InstanaAgentReconciler instance
func NewInstanaAgentRemoteReconciler(
	client client.Client,
	scheme *runtime.Scheme,
	recorder record.EventRecorder,
) *InstanaAgentRemoteReconciler {
	return &InstanaAgentRemoteReconciler{
		client:   instanaclient.NewInstanaAgentClient(client),
		recorder: recorder,
		scheme:   scheme,
	}
}

type InstanaAgentRemoteReconciler struct {
	client   instanaclient.InstanaAgentClient
	recorder record.EventRecorder
	scheme   *runtime.Scheme
}

func (r *InstanaAgentRemoteReconciler) reconcile(
	ctx context.Context,
	req ctrl.Request,
	statusManager status.InstanaAgentRemoteStatusManager,
) reconcileReturn {
	agent, getAgentRes := r.getInstanaAgentRemote(ctx, req)
	if getAgentRes.suppliesReconcileResult() {
		return getAgentRes
	}

	statusManager.SetAgentOld(agent)

	log := r.loggerFor(ctx, agent)
	ctx = logr.NewContext(ctx, log)
	log.Info("reconciling instana agent remote CR")

	agent.Default()

	operatorUtils := operator_utils.NewRemoteOperatorUtils(
		ctx,
		r.client,
		agent,
		lifecycle.NewRemoteDependentLifecycleManager(ctx, agent, r.client),
	)

	if handleDeletionRes := r.handleDeletion(ctx, agent, operatorUtils); handleDeletionRes.suppliesReconcileResult() {
		return handleDeletionRes
	}

	if addFinalizerRes := r.addOrUpdateFinalizers(ctx, agent); addFinalizerRes.suppliesReconcileResult() {
		return addFinalizerRes
	}

	keysSecret := &corev1.Secret{}
	if agent.Spec.Agent.KeysSecret != "" {
		if err := r.client.Get(ctx, client.ObjectKey{Name: agent.Spec.Agent.KeysSecret, Namespace: agent.Namespace}, keysSecret); err != nil {
			log.Error(err, "unable to get KeysSecret-field")
		}
	}

	backends := r.getRemoteSensorBackends(agent)

	if applyResourcesRes := r.applyResources(
		ctx,
		agent,
		operatorUtils,
		statusManager,
		keysSecret,
		backends,
	); applyResourcesRes.suppliesReconcileResult() {
		return applyResourcesRes
	}

	log.Info("successfully finished reconcile on Instna Agent Remote CR")

	return reconcileSuccess(ctrl.Result{})
}

// +kubebuilder:rbac:groups=instana.io,resources=agentsremote,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets;configmaps;services;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch
// +kubebuilder:rbac:groups=instana.io,resources=agentsremote/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=instana.io,resources=agentsremote/finalizers,verbs=update
// +kubebuilder:rbac:groups=policy,resources=podsecuritypolicies,verbs=use
// +kubebuilder:rbac:groups=security.openshift.io,resourceNames=anyuid,resources=securitycontextconstraints,verbs=use

func (r *InstanaAgentRemoteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (
	res ctrl.Result,
	reconcileErr error,
) {
	defer recovery.Catch(&reconcileErr)

	logger := logf.FromContext(ctx).WithName("instana-agent-remote-controller")
	ctx = logf.IntoContext(ctx, logger)

	statusManager := status.NewInstanaAgentRemoteStatusManager(r.client, r.recorder)
	defer func() {
		if err := statusManager.UpdateAgentStatus(ctx, reconcileErr); err != nil {
			errBuilder := multierror.NewMultiErrorBuilder(reconcileErr, err)
			reconcileErr = errBuilder.Build()
		}
	}()

	return r.reconcile(ctx, req, statusManager).reconcileResult()
}
