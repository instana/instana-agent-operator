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
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	instanaclient "github.com/instana/instana-agent-operator/pkg/k8s/client"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/lifecycle"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/operator_utils"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/status"
	"github.com/instana/instana-agent-operator/pkg/multierror"
	"github.com/instana/instana-agent-operator/pkg/recovery"
)

// Add will create a new Remote Agent Controller and add this to the Manager for reconciling
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
		Watches(
			&instanav1.InstanaAgent{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
				log := log.FromContext(ctx)

				// Ensure the triggering object is namespaced
				namespace := obj.GetNamespace()
				if namespace == "" {
					//agent needs to be namespaced bound. If not do no reconcile
					return nil
				}

				var remoteAgentList instanav1.RemoteAgentList
				if err := mgr.GetClient().List(ctx, &remoteAgentList, &client.ListOptions{
					Namespace: namespace,
				}); err != nil {
					//error retrieving remote agent specs in namespace. do not trigger reconcile
					return nil
				}

				//no remote agent specs in namespace. do not trigger reconcile
				if len(remoteAgentList.Items) == 0 {
					log.Info("No RemoteAgents in same namespace as InstanaAgent", "namespace", namespace)
					return nil
				}

				var requests []reconcile.Request
				for _, remoteAgent := range remoteAgentList.Items {
					requests = append(requests, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name:      remoteAgent.Name,
							Namespace: remoteAgent.Namespace,
						},
					})
				}
				return requests
			}),
		).
		WithEventFilter(filterPredicateRemote()).
		Complete(
			NewRemoteAgentReconciler(
				mgr.GetClient(),
				mgr.GetScheme(),
				mgr.GetEventRecorderFor("remote-agent-controller"),
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
	agent, getAgentRes := r.getRemoteAgent(ctx, req)
	if getAgentRes.suppliesReconcileResult() {
		return getAgentRes
	}

	statusManager.SetAgentOld(agent)

	log := r.loggerFor(ctx, agent)
	ctx = logr.NewContext(ctx, log)
	log.Info("reconciling remote Agent CR")

	hostAgent, _ := r.getAgent(ctx, agent.Namespace, "instana-agent")
	//if host agent exists in namespace inherit values otherwise default to minimum values to start an agent
	if hostAgent != nil {
		agent.DefaultWithHost(*hostAgent)
	} else {
		agent.Default()
	}

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

// +kubebuilder:rbac:groups=instana.io,resources=remoteagents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets;configmaps;services;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings,verbs=get;list;watch;create;update;patch;delete;bind
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch
// +kubebuilder:rbac:groups=instana.io,resources=remoteagents/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=instana.io,resources=remoteagents/finalizers,verbs=update
// +kubebuilder:rbac:groups=policy,resources=podsecuritypolicies,verbs=use

func (r *RemoteAgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (
	res ctrl.Result,
	reconcileErr error,
) {
	defer recovery.Catch(&reconcileErr)

	logger := logf.FromContext(ctx).WithName("remote-agent-controller")
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
