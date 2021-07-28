/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package controllers

import (
	"context"

	"k8s.io/client-go/rest"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/go-logr/logr"
	instanaV1Beta1 "github.com/instana/instana-agent-operator/api/v1beta1"
	"github.com/instana/instana-agent-operator/controllers/leaderelection"
	"github.com/instana/instana-agent-operator/controllers/reconciliation"

	appV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	instanaAgentFinalizer = "agent.instana.com/finalizer"
)

var (
	AppName        = "instana-agent"
	AgentNameSpace = AppName

	leaderElector *leaderelection.LeaderElector
)

// Add will create a new Instana Agent Controller and add this to the Manager for reconciling
func Add(mgr manager.Manager) error {
	return add(mgr, NewInstanaAgentReconciler(
		mgr.GetClient(),
		mgr.GetAPIReader(),
		mgr.GetScheme(),
		mgr.GetConfig(),
		logf.Log.WithName("agent.controller")))
}

// add sets up the controller with the Manager.
func add(mgr ctrl.Manager, r *InstanaAgentReconciler) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&instanaV1Beta1.InstanaAgent{}).
		Owns(&appV1.DaemonSet{}, builder.WithPredicates(filterPredicate())).
		Owns(&coreV1.Pod{}).
		Owns(&coreV1.Secret{}).
		Owns(&coreV1.ConfigMap{}).
		Owns(&coreV1.Service{}).
		Owns(&coreV1.ServiceAccount{}).
		Complete(r)
}

func filterPredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Evaluates to false if the object has been confirmed deleted.
			return !e.DeleteStateUnknown
		},
	}
}

// NewInstanaAgentReconciler initializes a new InstanaAgentReconciler instance
func NewInstanaAgentReconciler(client client.Client, apiReader client.Reader, scheme *runtime.Scheme, config *rest.Config, log logr.Logger) *InstanaAgentReconciler {
	return &InstanaAgentReconciler{
		client:              client,
		apiReader:           apiReader,
		scheme:              scheme,
		config:              config,
		log:                 log,
		agentReconciliation: reconciliation.New(client, scheme, log.WithName("reconcile")),
	}
}

type InstanaAgentReconciler struct {
	client              client.Client
	apiReader           client.Reader
	scheme              *runtime.Scheme
	config              *rest.Config
	log                 logr.Logger
	agentReconciliation reconciliation.Reconciliation
}

//+kubebuilder:rbac:groups=agents.instana.com,resources=instanaagent,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods;secrets;configmaps;services;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=agents.instana.com,resources=instanaagent/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=agents.instana.com,resources=instanaagent/finalizers,verbs=update
func (r *InstanaAgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()

	log := r.log.WithValues("namespace", req.Namespace, "name", req.Name)
	log.Info("Reconciling Instana Agent")

	crdInstance, err := r.fetchCrdInstance(ctx, req)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			r.log.Info("CRD object not found, could have been deleted after reconcile request")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	isInstanaAgentDeleted := crdInstance.GetDeletionTimestamp() != nil

	if isInstanaAgentDeleted {
		if controllerutil.ContainsFinalizer(crdInstance, instanaAgentFinalizer) {
			r.log.Info("Running the finalizer...")
			if err := r.finalizeAgent(crdInstance); err != nil {
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(crdInstance, instanaAgentFinalizer)
			err := r.client.Update(ctx, crdInstance)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(crdInstance, instanaAgentFinalizer) {
		controllerutil.AddFinalizer(crdInstance, instanaAgentFinalizer)
		err = r.client.Update(ctx, crdInstance)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if err = r.agentReconciliation.CreateOrUpdate(req, crdInstance); err != nil {
		return ctrl.Result{}, err
	}
	r.log.Info("Charts installed/upgraded successfully")

	if leaderelection.LeaderElectionTask == nil || leaderelection.LeaderElectionTask.IsCancelled() || leaderelection.LeaderElectionTaskScheduler.IsShutdown() {
		leaderElector = &leaderelection.LeaderElector{
			Ctx:    ctx,
			Client: r.client,
			Scheme: r.scheme,
		}
		leaderElector.StartCoordination(AgentNameSpace)
	}
	return ctrl.Result{}, nil
}

func (r *InstanaAgentReconciler) finalizeAgent(crdInstance *instanaV1Beta1.InstanaAgent) error {
	if err := r.agentReconciliation.Delete(crdInstance); err != nil {
		return err
	}
	if leaderElector != nil {
		leaderElector.CancelLeaderElection()
	}
	r.log.Info("Successfully finalized instana agent")
	return nil
}

func (r *InstanaAgentReconciler) fetchCrdInstance(ctx context.Context, req ctrl.Request) (*instanaV1Beta1.InstanaAgent, error) {
	crdInstance := &instanaV1Beta1.InstanaAgent{}
	// TODO use apiReader??
	err := r.client.Get(ctx, req.NamespacedName, crdInstance)
	if err != nil {
		return nil, err
	}
	r.log.Info("Reconciling Instana CRD")
	AppName = crdInstance.Name
	AgentNameSpace = crdInstance.Namespace
	return crdInstance, err
}
