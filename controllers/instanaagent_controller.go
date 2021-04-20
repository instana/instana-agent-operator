/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	instanaV1Beta1 "github.com/instana/instana-agent-operator/api/v1beta1"
	appV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
//+kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;

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
	_ = context.Background()
	_ = r.Log.WithValues("instanaagent", req.NamespacedName)

	// Fetch the InstanaAgent CRD instance
	crdInstance := &instanaV1Beta1.InstanaAgent{}
	err := r.Get(ctx, req.NamespacedName, crdInstance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}
	r.Log.Info("Instana CRD fetched successfully")
	//check if the daemonset already deployed, if not create a new one.
	foundDaemonset := &appV1.DaemonSet{}
	err = r.Get(ctx, req.NamespacedName, foundDaemonset)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("No Daemonset deployed before, creating new one")
			newDaemonset := newDaemonsetForCRD(crdInstance)
			r.Create(ctx, newDaemonset)
			r.Log.Info("Daemonset created successfully")
		}
	}

	return ctrl.Result{}, nil
}

// returns a Daemonset object with the data hold in instanaAgent crd instance
func newDaemonsetForCRD(crdInstance *instanaV1Beta1.InstanaAgent) *appV1.DaemonSet {
	//we need to have a same matched label for all our agent resources
	selectorLabels := labelsForApp()
	podSpec := newPodSpec(crdInstance)
	return &appV1.DaemonSet{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      crdInstance.Name,
			Namespace: crdInstance.Namespace,
			Labels:    selectorLabels,
		},
		Spec: appV1.DaemonSetSpec{
			Selector: &metaV1.LabelSelector{MatchLabels: selectorLabels},
			Template: coreV1.PodTemplateSpec{
				ObjectMeta: metaV1.ObjectMeta{
					Labels: selectorLabels,
				},
				Spec: podSpec,
			},
		},
	}
}

func newPodSpec(crdInstance *instanaV1Beta1.InstanaAgent) coreV1.PodSpec {
	envVars := crdInstance.Spec.Env
	agentEnvVars := []coreV1.EnvVar{
		{Name: "INSTANA_OPERATOR_MANAGED", Value: "true"},
		{Name: "INSTANA_AGENT_KEY", ValueFrom: &coreV1.EnvVarSource{
			SecretKeyRef: &coreV1.SecretKeySelector{
				LocalObjectReference: coreV1.LocalObjectReference{Name: "instana-agent"},
				Key:                  "key",
			},
		}},
		{Name: "INSTANA_AGENT_POD_NAME", ValueFrom: &coreV1.EnvVarSource{
			FieldRef: &coreV1.ObjectFieldSelector{
				FieldPath:  "metadata.name",
				APIVersion: "v1",
			},
		}},
		{Name: "POD_IP", ValueFrom: &coreV1.EnvVarSource{
			FieldRef: &coreV1.ObjectFieldSelector{
				FieldPath:  "status.podIP",
				APIVersion: "v1",
			},
		}},
	}

	mergedEnvVars := append(envVars, agentEnvVars...)
	return coreV1.PodSpec{
		Containers: []coreV1.Container{{
			Name:            "instana-agent",
			Image:           "instana/agent",
			ImagePullPolicy: coreV1.PullAlways,
			Env:             mergedEnvVars,
		}},
	}
}

func labelsForApp() map[string]string {
	return map[string]string{"app": "instana-agent"}
}

// SetupWithManager sets up the controller with the Manager.
func (r *InstanaAgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&instanaV1Beta1.InstanaAgent{}).
		Owns(&appV1.DaemonSet{}).
		Owns(&coreV1.Service{}).
		Owns(&coreV1.Pod{}).
		Complete(r)
}
