/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	instanaV1Beta1 "github.com/instana/instana-agent-operator/api/v1beta1"
	"github.com/pkg/errors"
	appV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	AppVersion               = "1.0.0-beta"
	AgentKey                 = "key"
	AgentDownloadKey         = "downloadKey"
	DefaultAgentImageName    = "instana/agent"
	AgentImagePullSecretName = "containers-key"
	DockerRegistry           = "containers.instana.io"

	AgentPort         = 42699
	OpenTelemetryPort = 55680
)

var (
	AppName                 = "instana-agent"
	AgentNameSpace          = AppName
	AgentSecretName         = AppName
	AgentServiceAccountName = AppName
)

type InstanaAgentReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=agents.instana.com,namespace=instana-agent,resources=instanaagent,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=agents.instana.com,namespace=instana-agent,resources=instanaagent/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=agents.instana.com,namespace=instana-agent,resources=instanaagent/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,namespace=instana-agent,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,namespace=instana-agent,resources=pods,verbs=get;list;
func (r *InstanaAgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("instanaagent", req.NamespacedName)

	crdInstance, err := r.fetchCrdInstance(ctx, req)
	if err != nil {
		return ctrl.Result{}, err
	} else if crdInstance == nil {
		r.Log.Error(errors.New("CRD not found"), "CRD object not found, could have been deleted after reconcile request")
		return ctrl.Result{}, nil
	}

	var reconcilationError = error(nil)

	if err = r.reconcileSecrets(ctx, crdInstance); err != nil {
		reconcilationError = err
	}
	if err = r.reconcileImagePullSecrets(ctx, crdInstance); err != nil {
		reconcilationError = err
	}
	if err = r.reconcileServices(ctx, crdInstance); err != nil {
		reconcilationError = err
	}
	if err = r.reconcileServiceAccounts(ctx, crdInstance); err != nil {
		reconcilationError = err
	}
	if err = r.reconcileConfigMap(ctx, crdInstance); err != nil {
		reconcilationError = err
	}
	if err = r.reconcileClusterRole(ctx); err != nil {
		reconcilationError = err
	}
	if err = r.reconcileClusterRoleBinding(ctx); err != nil {
		reconcilationError = err
	}
	if err = r.reconcileDaemonset(ctx, req, crdInstance); err != nil {
		reconcilationError = err
	}

	return ctrl.Result{}, reconcilationError
}

// SetupWithManager sets up the controller with the Manager.
func (r *InstanaAgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&instanaV1Beta1.InstanaAgent{}).
		Owns(&appV1.DaemonSet{}).
		Owns(&coreV1.Secret{}).
		Owns(&coreV1.Pod{}).
		Complete(r)
}

func (r *InstanaAgentReconciler) fetchCrdInstance(ctx context.Context, req ctrl.Request) (*instanaV1Beta1.InstanaAgent, error) {
	crdInstance := &instanaV1Beta1.InstanaAgent{}
	err := r.Get(ctx, req.NamespacedName, crdInstance)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return nil, nil
		}
		// Error reading the object - requeue the request.
		return nil, err
	}
	r.Log.Info("Reconciling Instana CRD")
	AppName = crdInstance.Name
	AgentNameSpace = crdInstance.Namespace
	return crdInstance, err
}

func buildLabels() map[string]string {
	return map[string]string{
		"app":                          AppName,
		"app.kubernetes.io/name":       AppName,
		"app.kubernetes.io/version":    AppVersion,
		"app.kubernetes.io/managed-by": AppName,
	}
}
