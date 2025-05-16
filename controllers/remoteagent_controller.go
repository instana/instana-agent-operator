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

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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

// Add will create a new Instana Agent Controller and add this to the Manager for reconciling
func AddRemote(mgr manager.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&instanav1.RemoteAgent{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&corev1.Service{}).
		Owns(&rbacv1.ClusterRole{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		WithEventFilter(filterPredicate()).
		Complete(
			NewRemoteAgentReconciler(
				mgr.GetClient(),
				mgr.GetScheme(),
				mgr.GetEventRecorderFor("remote-instana-agent-controller"),
			),
		)
}

// NewInstanaAgentReconciler initializes a new InstanaAgentReconciler instance
func NewRemoteAgentReconciler(
	client client.Client,
	scheme *runtime.Scheme,
	recorder record.EventRecorder,
) *RemoteAgentReconciler {
	return &RemoteAgentReconciler{
		client:   instanaclient.NewInstanaAgentClient(client),
		recorder: recorder,
		scheme:   scheme,
	}
}

type RemoteAgentReconciler struct {
	client   instanaclient.InstanaAgentClient
	recorder record.EventRecorder
	scheme   *runtime.Scheme
}

func (r *RemoteAgentReconciler) reconcile(
	ctx context.Context,
	req ctrl.Request,
	statusManager status.RemoteAgentStatusManager,
) reconcileReturn {
	agent, getAgentRes := r.getAgent(ctx, req)
	if getAgentRes.suppliesReconcileResult() {
		return getAgentRes
	}

	statusManager.SetAgentOld(agent)

	log := r.loggerFor(ctx, agent)
	ctx = logr.NewContext(ctx, log)
	log.Info("reconciling remote Agent CR")

	var hostAgent instanav1.InstanaAgent

	switch err := r.client.Get(ctx, req.NamespacedName, &hostAgent); {
	case k8serrors.IsNotFound(err):
		log.V(10).Info("attempted to reconcile agent CR that could not be found")
	case !errors.Is(err, nil):
		log.Error(err, "failed to retrieve info about agent CR")
	default:
		log.V(1).Info("successfully retrieved agent CR info")
	}

	agent.Default(hostAgent)

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

	k8SensorBackends := r.getK8SensorBackends(agent)

	if applyResourcesRes := r.applyResources(
		ctx,
		agent,
		operatorUtils,
		statusManager,
		keysSecret,
		k8SensorBackends,
	); applyResourcesRes.suppliesReconcileResult() {
		return applyResourcesRes
	}

	log.Info("successfully finished reconcile on remote agent CR")

	return reconcileSuccess(ctrl.Result{})
}

// +kubebuilder:rbac:groups=instana.io,resources=agents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets;configmaps;services;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings,verbs=get;list;watch;create;update;patch;delete;bind
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch
// +kubebuilder:rbac:groups=instana.io,resources=agents/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=instana.io,resources=agents/finalizers,verbs=update
// +kubebuilder:rbac:groups=policy,resources=podsecuritypolicies,verbs=use

// adding role property required to manage instana-agent-k8sensor ClusterRole
// +kubebuilder:rbac:urls=/version;/healthz;/metrics;/metrics/cadvisor;/stats/summary,verbs=get
// +kubebuilder:rbac:groups=extensions,resources=deployments;replicasets;ingresses,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps;events;services;endpoints;namespaces;nodes;pods;pods/log;replicationcontrollers;resourcequotas;persistentvolumes;persistentvolumeclaims;nodes/metrics;nodes/stats;nodes/proxy,verbs=get;list;watch
// +kubebuilder:rbac:groups=batch,resources=cronjobs;jobs,verbs=get;list;watch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps.openshift.io,resources=deploymentconfigs,verbs=get;list;watch
// +kubebuilder:rbac:groups=security.openshift.io,resourceNames=privileged,resources=securitycontextconstraints,verbs=use
// +kubebuilder:rbac:groups=policy,resourceNames=instana-agent-k8sensor,resources=podsecuritypolicies,verbs=use

func (r *RemoteAgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (
	res ctrl.Result,
	reconcileErr error,
) {
	defer recovery.Catch(&reconcileErr)

	logger := logf.FromContext(ctx).WithName("remote-instana-agent-controller")
	ctx = logf.IntoContext(ctx, logger)

	statusManager := status.NewRemoteAgentStatusManager(r.client, r.recorder)
	defer func() {
		if err := statusManager.UpdateAgentStatus(ctx, reconcileErr); err != nil {
			errBuilder := multierror.NewMultiErrorBuilder(reconcileErr, err)
			reconcileErr = errBuilder.Build()
		}
	}()

	return r.reconcile(ctx, req, statusManager).reconcileResult()
}
