/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
)

const (
	oldFinalizer = "agent.instana.io/finalizer"
)

// Add will create a new Instana Agent Controller and add this to the Manager for reconciling
func Add(mgr manager.Manager) error {
	return add(
		mgr, NewInstanaAgentReconciler(
			mgr.GetClient(),
			mgr.GetAPIReader(),
			mgr.GetEventRecorderFor("agent.controller"),
			mgr.GetScheme(),
			mgr.GetConfig(),
			logf.Log.WithName("agent.controller"),
		),
	)
}

// add sets up the controller with the Manager.
func add(mgr ctrl.Manager, r *InstanaAgentReconciler) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&instanav1.InstanaAgent{}).
		// TODO: Update list of Owns
		Owns(&appsv1.DaemonSet{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&corev1.Service{}).
		Owns(&rbacv1.ClusterRole{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		WithEventFilter(filterPredicate()).
		Complete(r)
}

// Create generic filter for all events, that removes some chattiness mainly when only the Status field has been updated.
func filterPredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change.
			return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Evaluates to false if the object has been confirmed deleted.
			return !e.DeleteStateUnknown
		},
	}
}

// NewInstanaAgentReconciler initializes a new InstanaAgentReconciler instance
func NewInstanaAgentReconciler(
	client client.Client,
	apiReader client.Reader,
	recorder record.EventRecorder,
	scheme *runtime.Scheme,
	config *rest.Config,
	log logr.Logger,
) *InstanaAgentReconciler {
	return &InstanaAgentReconciler{
		client:    client,
		apiReader: apiReader,
		recorder:  recorder,
		scheme:    scheme,
		config:    config,
		log:       log,
	}
}

type InstanaAgentReconciler struct {
	client    client.Client
	apiReader client.Reader
	recorder  record.EventRecorder
	scheme    *runtime.Scheme
	config    *rest.Config
	log       logr.Logger
}

// TODO: Update permissions here

// +kubebuilder:rbac:groups=agents.instana.io,resources=instanaagent,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods;secrets;configmaps;services;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=agents.instana.io,resources=instanaagent/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=agents.instana.io,resources=instanaagent/finalizers,verbs=update
func (r *InstanaAgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	panic("implement me") // TODO
}
