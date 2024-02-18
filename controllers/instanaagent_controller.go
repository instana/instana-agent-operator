/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package controllers

import (
	"context"
	"errors"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/controllers/reconciliation/helm"
	instanaclient "github.com/instana/instana-agent-operator/pkg/k8s/client"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/operator_utils"
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
			mgr.GetEventRecorderFor("agent.controller"),
			logf.Log.WithName("agent.controller"),
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

func wasModifiedByOther(obj client.Object) bool {
	var lastModifiedBySelf time.Time

	for _, mfe := range obj.GetManagedFields() {
		if mfe.Manager == instanaclient.FieldOwnerName {
			if mfe.Time == nil {
				return true
			}
			lastModifiedBySelf = mfe.Time.Time
			break
		}
	}

	if lastModifiedBySelf.IsZero() {
		return true
	}

	for _, mfe := range obj.GetManagedFields() {
		if mfe.Manager == instanaclient.FieldOwnerName {
			continue
		}
		if mfe.Time == nil {
			return true
		}
		if lastModifiedBySelf.Before(mfe.Time.Time) {
			return true
		}
	}

	return false
}

// Create generic filter for all events, that removes some chattiness mainly when only the Status field has been updated.
func filterPredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(createEvent event.CreateEvent) bool {
			switch createEvent.Object.(type) {
			case *instanav1.InstanaAgent:
				return true
			default:
				return false
			}
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			switch e.ObjectOld.(type) {
			case *instanav1.InstanaAgent:
				return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
			default:
				return wasModifiedByOther(e.ObjectNew)
			}
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			switch e.Object.(type) {
			case *instanav1.InstanaAgent:
				return !e.DeleteStateUnknown
			default:
				return true
			}
		},
	}
}

// NewInstanaAgentReconciler initializes a new InstanaAgentReconciler instance
func NewInstanaAgentReconciler(
	client client.Client,
	scheme *runtime.Scheme,
	recorder record.EventRecorder,
	log logr.Logger,
) *InstanaAgentReconciler {
	return &InstanaAgentReconciler{
		client:       instanaclient.NewClient(client),
		recorder:     recorder,
		log:          log,
		chartRemover: helm.NewHelmReconciliation(scheme, log),
	}
}

type InstanaAgentReconciler struct {
	client       instanaclient.InstanaAgentClient
	recorder     record.EventRecorder
	log          logr.Logger
	chartRemover helm.DeprecatedInternalChartUninstaller
}

func (r *InstanaAgentReconciler) getAgent(ctx context.Context, req ctrl.Request) (
	*instanav1.InstanaAgent,
	reconcileReturn,
) {
	var agent instanav1.InstanaAgent

	log := r.log.WithValues("Request", req)

	switch err := r.client.Get(ctx, req.NamespacedName, &agent); {
	case k8serrors.IsNotFound(err):
		log.V(10).Error(err, "attempted to reconcile agent CR that could not be found")
		return nil, reconcileSuccess(ctrl.Result{})
	case !errors.Is(err, nil):
		log.Error(err, "failed to retrieve info about agent CR")
		return nil, reconcileFailure(err)
	default:
		log.V(1).Info("successfully retrieved agent CR info")
		return &agent, reconcileContinue()
	}
}

func (r *InstanaAgentReconciler) updateAgent(ctx context.Context, agent *instanav1.InstanaAgent) reconcileReturn {
	log := r.loggerFor(agent)

	switch err := r.client.Update(ctx, agent); errors.Is(err, nil) {
	case true:
		log.V(1).Info("successfully applied updates to agent CR")
		return reconcileSuccess(ctrl.Result{Requeue: true})
	default:
		log.Error(err, "failed to apply updates to agent CR")
		return reconcileFailure(err)
	}
}

func (r *InstanaAgentReconciler) isOpenShift(operatorUtils operator_utils.OperatorUtils) (bool, reconcileReturn) {
	isOpenShiftRes := operatorUtils.ClusterIsOpenShift()
	answer, err := isOpenShiftRes.Get()

	switch isOpenShiftRes.IsSuccess() {
	case true:
		r.log.V(1).Info("successfully detected whether cluster is OpenShift", "IsOpenShift", answer)
		return answer, reconcileContinue()
	default:
		r.log.Error(err, "failed to determine if cluster is OpenShift")
		return false, reconcileFailure(err)
	}
}

func (r *InstanaAgentReconciler) loggerFor(agent *instanav1.InstanaAgent) logr.Logger {
	return r.log.WithValues("Name", agent.Name, "Namespace", agent.Namespace, "Generation", agent.Generation)
}

func (r *InstanaAgentReconciler) reconcile(ctx context.Context, req ctrl.Request) reconcileReturn {
	agent, getAgentRes := r.getAgent(ctx, req)
	if getAgentRes.suppliesReconcileResult() {
		return getAgentRes
	}

	log := r.loggerFor(agent)
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

	isOpenShift, isOpenShiftRes := r.isOpenShift(operatorUtils)
	if isOpenShiftRes.suppliesReconcileResult() {
		return isOpenShiftRes
	}

	if applyResourcesRes := r.applyResources(
		agent,
		isOpenShift,
		operatorUtils,
	); applyResourcesRes.suppliesReconcileResult() {
		return applyResourcesRes
	}

	log.Info("successfully finished reconcile on agent CR")

	return reconcileSuccess(ctrl.Result{}) // TODO: May or may not want to go again after some time for status updates
}

// +kubebuilder:rbac:groups=agents.instana.io,resources=instanaagent,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods;secrets;configmaps;services;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=agents.instana.io,resources=instanaagent/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=agents.instana.io,resources=instanaagent/finalizers,verbs=update
func (r *InstanaAgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	defer recovery.Catch(&err)

	return r.reconcile(ctx, req).reconcileResult()
}
