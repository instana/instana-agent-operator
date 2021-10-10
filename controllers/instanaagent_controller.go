/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package controllers

import (
	"context"
	"fmt"

	instanaV1 "github.com/instana/instana-agent-operator/api/v1"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/client-go/rest"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/go-logr/logr"
	"github.com/instana/instana-agent-operator/controllers/leaderelection"
	"github.com/instana/instana-agent-operator/controllers/reconciliation"

	appV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	instanaAgentFinalizer = "agent.instana.io/finalizer"
	crExpectedName        = "instana-agent"
	crExpectedNamespace   = "instana-agent"
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
		For(&instanaV1.InstanaAgent{}).
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
		agentReconciliation: reconciliation.New(scheme, log, crExpectedName, crExpectedNamespace),
		crAppName:           crExpectedName,
		crAppNamespace:      crExpectedNamespace,
	}
}

type InstanaAgentReconciler struct {
	client              client.Client
	apiReader           client.Reader
	scheme              *runtime.Scheme
	config              *rest.Config
	log                 logr.Logger
	agentReconciliation reconciliation.Reconciliation
	crAppName           string
	crAppNamespace      string
	// Uninitialized variables in NewInstanaAgentReconciler
	leaderElector *leaderelection.LeaderElector
}

//+kubebuilder:rbac:groups=agents.instana.io,resources=instanaagent,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods;secrets;configmaps;services;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=agents.instana.io,resources=instanaagent/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=agents.instana.io,resources=instanaagent/finalizers,verbs=update
func (r *InstanaAgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.log.WithValues("namespace", req.Namespace, "name", req.Name)
	log.Info("Reconciling Instana Agent")

	crdInstance, err := r.fetchAgentCrdInstance(ctx, req)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			r.log.Info("Instana Agent CRD instance not found, please install the InstanaAgent CustomResource")
			return ctrl.Result{}, nil
		} else {
			r.log.Error(err, "Failed to get Instana Agent CustomResource or invalid")
			return ctrl.Result{}, err
		}
	}

	isInstanaAgentDeleted := crdInstance.GetDeletionTimestamp() != nil
	if isInstanaAgentDeleted {
		r.log.Info("Instana Agent Operator CustomResource is deleted. Cleanup Agent.")

		if controllerutil.ContainsFinalizer(crdInstance, instanaAgentFinalizer) {
			// This is a kind of work-around. Normally should just directly execute the clean-up logic. But when the user removes
			// the entire instana-agent Namespace the Operator runtime will get deleted before having a chance to clean up.
			// Try to detect this and remove the Finalizer. Otherwise the user needs to manually remove the Finalizer to get
			// all garbage collected.
			// The proper way of cleaning up would be:
			// 1) remove the Operator Custom Resource
			// 2) remove everything else
			if instanaNamespace, err := r.fetchInstanaNamespace(ctx); err == nil && instanaNamespace.GetDeletionTimestamp() != nil {
				r.log.Info("Seems like the Instana namespace got deleted. Skip running the finalizer logic and try to remove finalizer.\n" +
					" Please delete the Instana Agent Operator CustomResource _first_!")
			} else {
				r.log.V(1).Info("Running the finalizer...")
				if err := r.finalizeAgent(req, crdInstance); err != nil {
					return ctrl.Result{}, err
				}
			}

			controllerutil.RemoveFinalizer(crdInstance, instanaAgentFinalizer)
			if err := r.client.Update(ctx, crdInstance); err != nil {
				return ctrl.Result{}, err
			}
			r.log.V(1).Info("Removed Finalizer from Instana Agent Operator CustomResource")
		}
		return ctrl.Result{}, nil
	}

	if err = r.getAndDeleteOldOperator(ctx); err != nil {
		r.log.Error(err, "Unrecoverable error updating the old Operator spec. Cannot continue Agent installation")
		return ctrl.Result{}, err
	}

	r.log.V(1).Info("Validating the CRD")
	if err = r.validateAgentCrd(crdInstance); err != nil {
		r.log.Error(err, "Unrecoverable error validating and converting the Instana Agent CRD for deployment")
		return ctrl.Result{}, err
	}

	//r.log.V(1).Info("Injecting finalizer into CRD, for cleanup when CRD gets removed")
	//if err = r.injectFinalizer(ctx, crdInstance); err != nil {
	//	return ctrl.Result{}, err
	//}

	// First try to start Leader Election Coordination so to return error if we cannot get it started
	if r.leaderElector == nil || !r.leaderElector.IsLeaderElectionScheduled() {
		if r.leaderElector != nil {
			// As we'll replace the Leader Elector instance make sure to properly clean up old one
			r.leaderElector.CancelLeaderElection()
		}

		r.leaderElector = leaderelection.NewLeaderElection(r.client)
		if err = r.leaderElector.StartCoordination(r.crAppNamespace); err != nil {
			r.log.Error(err, "Failure starting Leader Election Coordination")
			return ctrl.Result{}, err
		}
	}

	//crdInstance.Status.
	//err = r.client.Update(ctx, crdInstance)
	//if err != nil {
	//	log.Error(err, "Failed to update Memcached status")
	//	return ctrl.Result{}, err
	//}

	if err = r.agentReconciliation.CreateOrUpdate(req, crdInstance); err != nil {
		return ctrl.Result{}, err
	}
	r.log.Info("Agent installed/upgraded successfully")

	return ctrl.Result{}, nil
}

func (r *InstanaAgentReconciler) finalizeAgent(req ctrl.Request, crdInstance *instanaV1.InstanaAgent) error {
	if err := r.agentReconciliation.Delete(req, crdInstance); err != nil {
		return err
	}
	if r.leaderElector != nil {
		r.leaderElector.CancelLeaderElection()
	}
	r.log.Info("Successfully finalized instana agent")
	return nil
}

func (r *InstanaAgentReconciler) injectFinalizer(ctx context.Context, o client.Object) error {
	if !controllerutil.ContainsFinalizer(o, instanaAgentFinalizer) {
		controllerutil.AddFinalizer(o, instanaAgentFinalizer)
		return r.client.Update(ctx, o)
	}
	return nil
}

func (r *InstanaAgentReconciler) fetchAgentCrdInstance(ctx context.Context, req ctrl.Request) (*instanaV1.InstanaAgent, error) {
	crdInstance := &instanaV1.InstanaAgent{}
	if err := r.client.Get(ctx, req.NamespacedName, crdInstance); err != nil {
		return nil, err
	}

	// Verify if the CR has the expected Name / Namespace set. At a later time we could really make this configurable and install
	// our Agent in the given Namespace. For now, we only support the fixed value.
	if crExpectedName != crdInstance.Name || crExpectedNamespace != crdInstance.Namespace {
		err := fmt.Errorf("Instana Agent CustomResource Name (%v) or Namespace (%v) don't match currently mandatory Name='%v' and Namespace='%v'. Please adjust the CustomResource",
			crdInstance.Name, crdInstance.Namespace, crExpectedName, crExpectedNamespace)
		return nil, err
	}

	r.log.V(1).Info(fmt.Sprintf("Found Instana CustomResource: %v", crdInstance))
	return crdInstance, nil
}

// fetchInstanaNamespace will get the Namespace instance for ourselves
func (r *InstanaAgentReconciler) fetchInstanaNamespace(ctx context.Context) (*coreV1.Namespace, error) {
	instanaNamespace := &coreV1.Namespace{}
	if err := r.client.Get(ctx, client.ObjectKey{
		Namespace: "",
		Name:      crExpectedNamespace,
	}, instanaNamespace); err != nil {
		return nil, err
	}

	r.log.V(1).Info(fmt.Sprintf("Found Instana-Agent Namespace: %v", instanaNamespace))
	return instanaNamespace, nil
}

func (r *InstanaAgentReconciler) getAndDeleteOldOperator(ctx context.Context) error {
	oldOperatorDeployment := &appV1.Deployment{}
	if err := r.client.Get(ctx, client.ObjectKey{
		Namespace: "instana-agent",
		Name:      "instana-agent-operator",
	}, oldOperatorDeployment); err != nil {
		if k8sErrors.IsNotFound(err) {
			r.log.V(1).Info("No old Operator Deployment found, not necessary to delete")
			return nil
		} else {
			r.log.Error(err, "Failure looking for old Operator Deployment")
			return err
		}
	}

	r.log.V(1).Info(fmt.Sprintf("Found old Operator Deployment and will try to delete: %v", oldOperatorDeployment))
	if err := r.client.Delete(ctx, oldOperatorDeployment); err != nil {
		r.log.Error(err, "Failure deleting old Operator Deployment")
		return err
	}

	return nil
}

// validateAgentCrd does some basic validation as otherwise Helm may not deploy the Agent DaemonSet but silently skip it if
// certain fields are omitted. In the future we should prevent this by adding a Validation WebHook.
func (r *InstanaAgentReconciler) validateAgentCrd(crd *instanaV1.InstanaAgent) error {
	if len(crd.Spec.Agent.EndpointHost) == 0 || len(crd.Spec.Agent.EndpointPort) == 0 || crd.Spec.Agent.EndpointPort == "0" {
		r.log.Info(`
##############################################################################
####    ERROR: You did not specify a correct Endpoint (host and/or port)  ####
##############################################################################
`)
		return fmt.Errorf("CRD Agent Spec should contain valid EndpointHost and EndpointPort")
	}

	if len(crd.Spec.Cluster.Name) == 0 && len(crd.Spec.Zone.Name) == 0 {
		r.log.Info(`
##############################################################################
####    ERROR: You did not specify a zone or name for this cluster.       ####
##############################################################################
`)
		return fmt.Errorf("CRD Agent Spec should contain either Zone or Cluster name")
	}

	if len(crd.Spec.Agent.Key) == 0 && len(crd.Spec.Agent.KeysSecret) == 0 {
		r.log.Info(`
##############################################################################
####    ERROR: You did not specify your secret agent key.                 ####
##############################################################################
`)
		return fmt.Errorf("CRD Agent Spec should contain either Key or KeySecret")
	}

	return nil
}
