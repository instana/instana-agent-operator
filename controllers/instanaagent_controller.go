/*
(c) Copyright IBM Corp. 2024, 2025

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
	v1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
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

const (
	finalizerV1 = "agent.instana.io/finalizer"
	finalizerV3 = "v3.agent.instana.io/finalizer"
)

// Add will create a new Instana Agent Controller and add this to the Manager for reconciling
func Add(mgr manager.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&instanav1.InstanaAgent{}).
		Watches(
			&corev1.Namespace{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, a client.Object) []ctrl.Request {
				// Get all InstanaAgent custom resources and trigger reconciliation for each one
				var agentList instanav1.InstanaAgentList
				if err := mgr.GetClient().List(ctx, &agentList); err != nil {
					// Log error but don't block execution
					log.Log.Error(err, "Failed to list InstanaAgent resources")
					return []ctrl.Request{}
				}

				// Create reconciliation requests for each InstanaAgent CR
				var requests []ctrl.Request
				for _, agent := range agentList.Items {
					requests = append(requests, ctrl.Request{
						NamespacedName: types.NamespacedName{
							Name:      agent.GetName(),
							Namespace: agent.GetNamespace(),
						},
					})
				}

				return requests
			}),
		).
		Owns(&appsv1.DaemonSet{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&corev1.Service{}).
		Owns(&v1.PodDisruptionBudget{}).
		Owns(&rbacv1.ClusterRole{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		WithEventFilter(filterPredicate()).
		Complete(
			NewInstanaAgentReconciler(
				mgr.GetClient(),
				mgr.GetScheme(),
				mgr.GetEventRecorderFor("agent-controller"),
			),
		)
}

// NewInstanaAgentReconciler initializes a new InstanaAgentReconciler instance
func NewInstanaAgentReconciler(
	client client.Client,
	scheme *runtime.Scheme,
	recorder record.EventRecorder,
) *InstanaAgentReconciler {
	return &InstanaAgentReconciler{
		client:   instanaclient.NewInstanaAgentClient(client),
		recorder: recorder,
		scheme:   scheme,
	}
}

type InstanaAgentReconciler struct {
	client   instanaclient.InstanaAgentClient
	recorder record.EventRecorder
	scheme   *runtime.Scheme
}

func (r *InstanaAgentReconciler) reconcile(
	ctx context.Context,
	req ctrl.Request,
	statusManager status.AgentStatusManager,
) reconcileReturn {
	agent, getAgentRes := r.getAgent(ctx, req)
	if getAgentRes.suppliesReconcileResult() {
		return getAgentRes
	}

	statusManager.SetAgentOld(agent)

	log := r.loggerFor(ctx, agent)
	ctx = logr.NewContext(ctx, log)
	log.Info("reconciling Agent CR")

	agent.Default()
	operatorUtils := operator_utils.NewOperatorUtils(
		ctx,
		r.client,
		agent,
		lifecycle.NewDependentLifecycleManager(ctx, agent, r.client),
	)

	if handleDeletionRes := r.handleDeletion(ctx, agent, operatorUtils); handleDeletionRes.suppliesReconcileResult() {
		return handleDeletionRes
	}

	if addFinalizerRes := r.addOrUpdateFinalizers(ctx, agent); addFinalizerRes.suppliesReconcileResult() {
		return addFinalizerRes
	}

	isOpenShift, isOpenShiftRes := r.isOpenShift(ctx, operatorUtils)
	if isOpenShiftRes.suppliesReconcileResult() {
		return isOpenShiftRes
	}

	keysSecret := &corev1.Secret{}
	if agent.Spec.Agent.KeysSecret != "" {
		if err := r.client.Get(ctx, client.ObjectKey{Name: agent.Spec.Agent.KeysSecret, Namespace: agent.Namespace}, keysSecret); err != nil {
			log.Error(err, "unable to get KeysSecret-field")
		}
	}

	k8SensorBackends := r.getK8SensorBackends(agent)

	namespacesList, err := r.client.GetNamespacesWithLabels(ctx)
	if err != nil {
		log.Error(err, "unable to fetch list of namespaces with labels")
	}

	if applyResourcesRes := r.applyResources(
		ctx,
		agent,
		isOpenShift,
		operatorUtils,
		statusManager,
		keysSecret,
		k8SensorBackends,
		namespacesList,
	); applyResourcesRes.suppliesReconcileResult() {
		return applyResourcesRes
	}

	log.Info("successfully finished reconcile on agent CR")

	return reconcileSuccess(ctrl.Result{})
}

// +kubebuilder:rbac:groups=instana.io,resources=agents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=daemonsets;deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets;configmaps;services;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings,verbs=get;list;watch;create;update;patch;delete;bind
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch
// +kubebuilder:rbac:groups=instana.io,resources=agents/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=instana.io,resources=agents/finalizers,verbs=update
// +kubebuilder:rbac:groups=policy,resources=podsecuritypolicies,verbs=use

// adding role property required to manage instana-agent-k8sensor ClusterRole
// +kubebuilder:rbac:urls=/version;/healthz;/metrics;/metrics/cadvisor;/stats/summary,verbs=get
// +kubebuilder:rbac:groups=extensions,resources=deployments;replicasets;ingresses,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=configmaps;events;services;endpoints;namespaces;nodes;pods;pods/log;replicationcontrollers;resourcequotas;persistentvolumes;persistentvolumeclaims;nodes/metrics;nodes/stats;nodes/proxy,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=daemonsets;deployments;replicasets;statefulsets,verbs=get;list;watch
// +kubebuilder:rbac:groups=batch,resources=cronjobs;jobs,verbs=get;list;watch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps.openshift.io,resources=deploymentconfigs,verbs=get;list;watch
// +kubebuilder:rbac:groups=security.openshift.io,resourceNames=privileged,resources=securitycontextconstraints,verbs=use
// +kubebuilder:rbac:groups=policy,resourceNames=instana-agent-k8sensor,resources=podsecuritypolicies,verbs=use
// +kubebuilder:rbac:groups=policy,resourceNames=instana-agent-k8sensor,resources=poddisruptionbudgets,verbs=create;patch;delete
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=watch;list

func (r *InstanaAgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (
	res ctrl.Result,
	reconcileErr error,
) {
	logger := logf.FromContext(ctx).WithName("agent-controller")
	ctx = logf.IntoContext(ctx, logger)

	agent := &instanav1.InstanaAgent{}
	err := r.client.Get(ctx, req.NamespacedName, agent)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected.
			// Return and don't requeue
			logger.Info("InstanaAgent resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		logger.Error(err, "Failed to get InstanaAgent")
	}
	defer recovery.Catch(&reconcileErr)

	statusManager := status.NewAgentStatusManager(r.client, r.recorder)
	defer func() {
		if err := statusManager.UpdateAgentStatus(ctx, reconcileErr); err != nil {
			errBuilder := multierror.NewMultiErrorBuilder(reconcileErr, err)
			reconcileErr = errBuilder.Build()
		}
	}()

	return r.reconcile(ctx, req, statusManager).reconcileResult()
}
