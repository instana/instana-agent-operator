/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	instanav1beta1 "github.com/instana/instana-agent-operator/api/v1beta1"
	appV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// InstanaAgentReconciler reconciles a InstanaAgent object
type InstanaAgentReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=agents.instana.com,resources=instanaagent,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=agents.instana.com,resources=instanaagent/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=agents.instana.com,resources=instanaagent/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the InstanaAgent object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *InstanaAgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("instanaagent", req.NamespacedName)

	// your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *InstanaAgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&instanav1beta1.InstanaAgent{}).
		Owns(&appV1.DaemonSet{}).
		Owns(&coreV1.Service{}).
		Owns(&coreV1.Pod{}).
		Complete(r)
}
