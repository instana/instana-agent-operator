/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package controllers

import (
	"context"
	"fmt"
	"math"
	"time"

	"k8s.io/client-go/tools/record"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "k8s.io/api/rbac/v1"

	"github.com/blang/semver"

	"github.com/google/go-cmp/cmp"

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

var (
	minimumSupportedChartVersion, _ = semver.Parse("1.2.28")
)

// Add will create a new Instana Agent Controller and add this to the Manager for reconciling
func Add(mgr manager.Manager) error {
	return add(mgr, NewInstanaAgentReconciler(
		mgr.GetClient(),
		mgr.GetAPIReader(),
		mgr.GetEventRecorderFor("agent.controller"),
		mgr.GetScheme(),
		mgr.GetConfig(),
		logf.Log.WithName("agent.controller")))
}

// add sets up the controller with the Manager.
func add(mgr ctrl.Manager, r *InstanaAgentReconciler) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&instanaV1.InstanaAgent{}).
		Owns(&appV1.DaemonSet{}).
		Owns(&coreV1.Pod{}).
		Owns(&coreV1.ConfigMap{}).
		Owns(&coreV1.Secret{}).
		Owns(&coreV1.ServiceAccount{}).
		Owns(&coreV1.Service{}).
		Owns(&v1.ClusterRole{}).
		Owns(&v1.ClusterRoleBinding{}).
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
func NewInstanaAgentReconciler(client client.Client, apiReader client.Reader, recorder record.EventRecorder, scheme *runtime.Scheme, config *rest.Config, log logr.Logger) *InstanaAgentReconciler {
	return &InstanaAgentReconciler{
		client:              client,
		apiReader:           apiReader,
		recorder:            recorder,
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
	recorder            record.EventRecorder
	scheme              *runtime.Scheme
	config              *rest.Config
	log                 logr.Logger
	agentReconciliation reconciliation.Reconciliation
	crAppName           string
	crAppNamespace      string
	// Uninitialized variables in NewInstanaAgentReconciler
	leaderElector *leaderelection.LeaderElector
}

// +kubebuilder:rbac:groups=agents.instana.io,resources=instanaagent,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods;secrets;configmaps;services;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=agents.instana.io,resources=instanaagent/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=agents.instana.io,resources=instanaagent/finalizers,verbs=update
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
					return r.handleError(ctx, req, err, crdInstance)
				}
			}

			controllerutil.RemoveFinalizer(crdInstance, instanaAgentFinalizer)
			if err := r.client.Update(ctx, crdInstance); err != nil {
				return r.handleError(ctx, req, err, crdInstance)
			}
			r.log.V(1).Info("Removed Finalizer from Instana Agent Operator CustomResource")
		}
		return ctrl.Result{}, nil
	}

	// Validate the Custom Resource object (configuration) before we're taking any other actions
	r.log.V(1).Info("Validating the CRD")
	if err := r.validateAgentCrd(crdInstance); err != nil {
		r.log.Error(err, "Unrecoverable error validating the Instana Agent CRD for deployment")
		return r.handleError(ctx, req, err, crdInstance)
	}

	if !crdInstance.Status.OldVersionsUpdated {
		// If something got deleted, give the Operator another reconcile loop to clean up before continuing. So return immediately
		if deleted, err := r.purgeOldResources(ctx); err != nil {
			return r.handleError(ctx, req, err, crdInstance)
		} else if deleted {
			return ctrl.Result{RequeueAfter: time.Second * 1}, nil
		}

		if err := r.upsertCrdStatusFields(ctx, req, instanaV1.OperatorStateUpdating, func(status *instanaV1.InstanaAgentStatus) instanaV1.InstanaAgentStatus {
			status.OldVersionsUpdated = true
			return *status
		}); err != nil {
			if k8sErrors.IsConflict(err) {
				// do manual retry without error
				r.log.V(1).Info("CRD update conflict, rescheduling...")
				return ctrl.Result{RequeueAfter: time.Second * 1}, nil
			}
			r.log.Error(err, "Failed to update Instana Agent CRD Status field - old versions purged")
			return ctrl.Result{}, err
		}
	}

	//
	// Potential Old Operator resources removed, start installation of (new) Operator
	//

	r.log.V(1).Info("Injecting finalizer into CRD, for cleanup when CRD gets removed")
	if err := r.injectFinalizer(ctx, req, crdInstance); err != nil {
		if k8sErrors.IsConflict(err) {
			// do manual retry without error
			r.log.V(1).Info("CRD update conflict, rescheduling...")
			return ctrl.Result{RequeueAfter: time.Second * 1}, nil
		}
		r.log.Error(err, "Failure adding finalizer into CRD")
		return r.handleError(ctx, req, err, crdInstance)
	}

	// First try to start Leader Election Coordination so to return error if we cannot get it started
	if (r.leaderElector == nil || !r.leaderElector.IsLeaderElectionScheduled()) && !crdInstance.Spec.K8sSensor.DeploymentSpec.Enabled.Enabled {
		if r.leaderElector != nil {
			// As we'll replace the Leader Elector instance make sure to properly clean up old one
			r.leaderElector.CancelLeaderElection()
		}

		r.leaderElector = leaderelection.NewLeaderElection(r.client, req.NamespacedName)
		if err := r.leaderElector.StartCoordination(r.crAppNamespace); err != nil {
			r.log.Error(err, "Failure starting Leader Election Coordination")
			return r.handleError(ctx, req, err, crdInstance)
		}
	}

	if err := r.agentReconciliation.CreateOrUpdate(req, crdInstance); err != nil {
		return r.handleError(ctx, req, err, crdInstance)
	}
	r.log.Info("Agent installed/upgraded successfully")

	if err := r.updateStatusFieldsAndFireEvent(ctx, req, crdInstance); err != nil {
		if k8sErrors.IsConflict(err) {
			// do manual retry without error
			r.log.V(1).Info("CRD update conflict, rescheduling...")
			return ctrl.Result{RequeueAfter: time.Second * 1}, nil
		}
		r.log.Error(err, "Failed to update Instana Agent CRD Status field - resource references")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// handleError will set the Operator Status to a Failure state with a description of the error, and generate an Event for this
// error. Furthermore, it will increase the retry interval as otherwise the reconcile loop might get repeated in fast succession
// without chance of resolving the issue. Kubernetes will back-off by itself, but initially with shorter intervals and might not
// correctly work with different errors.
func (r *InstanaAgentReconciler) handleError(ctx context.Context, req ctrl.Request, originalErr error, crdInstance *instanaV1.InstanaAgent) (ctrl.Result, error) {
	var retryInterval time.Duration
	lastUpdate := crdInstance.Status.LastUpdate.Time
	lastStatus := crdInstance.Status.Status
	if lastUpdate.IsZero() || lastStatus != instanaV1.OperatorStateFailed {
		// If not already in a 'failure' state, retry after one second.
		retryInterval = time.Second
	} else {
		// Already in failure state, base the retry interval on the previous update interval
		retryInterval = time.Since(lastUpdate).Round(time.Second)
	}

	r.recorder.Event(crdInstance, "Warning", "ProcessingError", originalErr.Error())

	if err := r.upsertCrdStatusFields(ctx, req, instanaV1.OperatorStateFailed, func(status *instanaV1.InstanaAgentStatus) instanaV1.InstanaAgentStatus {
		status.Reason = originalErr.Error()
		return *status
	}); err != nil {
		// Cannot update status, so retry immediately.
		r.log.Error(err, fmt.Sprintf("unable to update status after handling error: %v", originalErr))
		return ctrl.Result{RequeueAfter: time.Second * 1}, nil
	}

	return ctrl.Result{
		// Keep doubling the interval with a max of 6 hours
		RequeueAfter: time.Duration(math.Min(float64(retryInterval.Nanoseconds()*2), float64(time.Hour.Nanoseconds()*6))),
		Requeue:      true,
	}, nil
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

func (r *InstanaAgentReconciler) injectFinalizer(ctx context.Context, req ctrl.Request, crdInstance *instanaV1.InstanaAgent) error {
	if !controllerutil.ContainsFinalizer(crdInstance, instanaAgentFinalizer) {
		// Pull the CR object again, so we're sure to have the latest version including changes
		if err := r.client.Get(ctx, req.NamespacedName, crdInstance); err != nil {
			return err
		}

		controllerutil.AddFinalizer(crdInstance, instanaAgentFinalizer)
		return r.client.Update(ctx, crdInstance)
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

// validateAgentCrd does some basic validation as otherwise Helm may not deploy the Agent DaemonSet but silently skip it if
// certain fields are omitted.
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

	if len(crd.Spec.PinnedChartVersion) > 0 {
		if pinnedChartVersion, err := semver.Parse(crd.Spec.PinnedChartVersion); err != nil {
			r.log.Info(`
##############################################################################
####    ERROR: Invalid Helm Chart version pinned, cannot be parsed.       ####
##############################################################################
`)
			return fmt.Errorf("Helm Chart version pinned but invalid format")

		} else if pinnedChartVersion.Compare(minimumSupportedChartVersion) < 0 {
			r.log.Info(fmt.Sprintf(`
##############################################################################
####    ERROR: Invalid Helm Chart version pinned.                         ####
####    Pick a version %v or higher.                                  ####
##############################################################################
`, minimumSupportedChartVersion))
			return fmt.Errorf("Helm Chart version pinned but does not meet minimum supported version")
		}
	}

	return nil
}

func (r *InstanaAgentReconciler) purgeOldResources(ctx context.Context) (bool, error) {
	r.log.V(1).Info("Checking for old Agent Operator installations and purging / upgrading them")

	if deleted, err := r.getAndDeleteOldOperator(ctx); err != nil {
		r.log.Error(err, "Unrecoverable error removing the old Operator Deployment spec. Cannot continue Agent installation")
		return false, err
	} else if deleted {
		return true, nil
	}

	if deleted, err := r.getAndDeleteOldOperatorResources(ctx); err != nil {
		r.log.Error(err, "Unrecoverable error updating old resources for Helm-based installation. Cannot continue Agent installation")
		return false, err
	} else if deleted {
		return true, nil
	}

	return false, nil
}

func (r *InstanaAgentReconciler) upsertCrdStatusFields(ctx context.Context, req ctrl.Request, resultState instanaV1.AgentOperatorState, statusFn func(status *instanaV1.InstanaAgentStatus) instanaV1.InstanaAgentStatus) error {
	// Pull the CR object again, so we're sure to have the latest version including changes
	crdInstance := &instanaV1.InstanaAgent{}
	if err := r.client.Get(ctx, req.NamespacedName, crdInstance); err != nil {
		return err
	}

	// Always update timestamp and state. Do this before executing statusFn so if _must_, it can be overwritten
	crdInstance.Status.Status = resultState
	crdInstance.Status.Reason = ""
	crdInstance.Status.LastUpdate = metav1.Now()

	crdInstance.Status = statusFn(&crdInstance.Status)

	return r.client.Status().Update(ctx, crdInstance)
}

func (r *InstanaAgentReconciler) updateStatusFieldsAndFireEvent(ctx context.Context, req ctrl.Request, crdInstance *instanaV1.InstanaAgent) error {
	r.log.V(1).Info("Updating Agent CRD Status field with references to DaemonSet and ConfigMap")

	configMaps := &coreV1.ConfigMapList{}
	if err := r.client.List(ctx, configMaps, client.InNamespace(r.crAppNamespace)); err != nil {
		r.log.Error(err, "Failed getting ConfigMap to update Instana Agent CRD Status field")
		return err
	}
	var configMapResource instanaV1.ResourceInfo
	for _, val := range configMaps.Items {
		if val.Name == "instana-agent" {
			configMapResource = instanaV1.ResourceInfo{
				Name: val.Name,
				UID:  string(val.UID),
			}
		}
	}

	daemonSets := &appV1.DaemonSetList{}
	if err := r.client.List(ctx, daemonSets, client.InNamespace(r.crAppNamespace)); err != nil {
		r.log.Error(err, "Failed getting DaemonSet to update Instana Agent CRD Status field")
		return err
	}
	var daemonSetResource instanaV1.ResourceInfo
	for _, val := range daemonSets.Items {
		if val.Name == "instana-agent" {
			daemonSetResource = instanaV1.ResourceInfo{
				Name: val.Name,
				UID:  string(val.UID),
			}
		}
	}

	if !cmp.Equal(configMapResource, crdInstance.Status.ConfigMap) ||
		!cmp.Equal(daemonSetResource, crdInstance.Status.DaemonSet) {

		r.recorder.Event(crdInstance, "Normal", "CreatedUpdated", "The Agent deployment was correctly created / updated")

		state := instanaV1.OperatorStateRunning
		if (instanaV1.ResourceInfo{}) == configMapResource || (instanaV1.ResourceInfo{}) == daemonSetResource {
			state = instanaV1.OperatorStateUpdating
		}

		return r.upsertCrdStatusFields(ctx, req, state, func(status *instanaV1.InstanaAgentStatus) instanaV1.InstanaAgentStatus {
			status.ConfigMap = configMapResource
			status.DaemonSet = daemonSetResource
			return *status
		})
	}

	return nil
}
