/*
 * (c) Copyright IBM Corp. 2021, 2024
 * (c) Copyright Instana Inc. 2021, 2024
 */

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	instanaclient "github.com/instana/instana-agent-operator/pkg/k8s/client"
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
	return add(
		mgr, NewInstanaAgentReconciler(
			mgr.GetClient(),
			mgr.GetScheme(),
			mgr.GetEventRecorderFor("agent-controller"),
		),
	)
}

// add sets up the controller with the Manager.
func add(mgr ctrl.Manager, r *InstanaAgentReconciler) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&instanav1.InstanaAgent{}).
		Owns(&appsv1.DaemonSet{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&corev1.Service{}).
		Owns(&rbacv1.ClusterRole{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		WithEventFilter(filterPredicate()).
		Complete(r)
}

// NewInstanaAgentReconciler initializes a new InstanaAgentReconciler instance
func NewInstanaAgentReconciler(
	client client.Client,
	scheme *runtime.Scheme,
	recorder record.EventRecorder,
) *InstanaAgentReconciler {
	return &InstanaAgentReconciler{
		client:   instanaclient.NewClient(client),
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
	operatorUtils := operator_utils.NewOperatorUtils(ctx, r.client, agent)

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

	if agent.Spec.Agent.KeysSecret == "" {
		log.V(1).Info("No Agent KeysSecret defined, no need to retrieve a secret from Kubernetes")
	} else {
		secretName := agent.Spec.Agent.KeysSecret
		namespace := agent.Namespace

		if err := r.client.Get(ctx, client.ObjectKey{Name: secretName, Namespace: namespace}, keysSecret); err != nil {
			log.Error(err, "Failed to fetch KeysSecret")
		} else {
			// secretData := secretKey.Data["key"]
			// fmt.Printf("Secret value: %s\n", string(secretData))
			fmt.Printf("Secret value found")
		}
	}

	if applyResourcesRes := r.applyResources(
		ctx,
		agent,
		isOpenShift,
		operatorUtils,
		statusManager,
		keysSecret,
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

// adding role property required to manage instana-agent-k8sensor ClusterRole
// +kubebuilder:rbac:urls=/version;/healthz,verbs=get
// +kubebuilder:rbac:groups=extensions,resources=deployments;replicasets;ingresses,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=configmaps;events;services;endpoints;namespaces;nodes;pods;pods/log;replicationcontrollers;resourcequotas;persistentvolumes;persistentvolumeclaims,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=daemonsets;deployments;replicasets;statefulsets,verbs=get;list;watch
// +kubebuilder:rbac:groups=batch,resources=cronjobs;jobs,verbs=get;list;watch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps.openshift.io,resources=deploymentconfigs,verbs=get;list;watch
// +kubebuilder:rbac:groups=security.openshift.io,resourceNames=privileged,resources=securitycontextconstraints,verbs=use
// +kubebuilder:rbac:groups=policy,resourceNames=instana-agent-k8sensor,resources=podsecuritypolicies,verbs=use

func (r *InstanaAgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (
	res ctrl.Result,
	reconcileErr error,
) {
	defer recovery.Catch(&reconcileErr)

	logger := logf.FromContext(ctx).WithName("agent-controller")
	ctx = logf.IntoContext(ctx, logger)

	statusManager := status.NewAgentStatusManager(r.client, r.recorder)
	defer func() {
		if err := statusManager.UpdateAgentStatus(ctx, reconcileErr); err != nil {
			errBuilder := multierror.NewMultiErrorBuilder(reconcileErr, err)
			reconcileErr = errBuilder.Build()
		}
	}()

	return r.reconcile(ctx, req, statusManager).reconcileResult()
}
