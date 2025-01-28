/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	agentoperatorv1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/controllers"
	"github.com/instana/instana-agent-operator/version"
	// +kubebuilder:scaffold:imports
)

var (
	scheme = k8sruntime.NewScheme()
	log    = logf.Log.WithName("main")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(agentoperatorv1.AddToScheme(scheme))
	utilruntime.Must(appsv1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var probeAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	// By default disable leader-election and assume single instance gets installed. Via parameters (--leader-elect) it will be
	// enabled from the Operator Deployment spec.
	flag.BoolVar(
		&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.",
	)

	// When running in debug-mode, include some more logging etc
	debugMode, _ := strconv.ParseBool(os.Getenv("DEBUG_MODE"))

	opts := zap.Options{
		Development: debugMode,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	// Set the Logger to be used also by the controller-runtime
	logf.SetLogger(zap.New(zap.UseFlagOptions(&opts)).WithName("instana"))

	printVersion()
	cfg := ctrl.GetConfigOrDie()

	mgr, err := ctrl.NewManager(
		cfg, ctrl.Options{
			Metrics: metricsserver.Options{
				BindAddress: metricsAddr,
			},
			Scheme:                 scheme,
			HealthProbeBindAddress: probeAddr,
			LeaderElection:         enableLeaderElection,
			LeaderElectionID:       "819a9291.instana.io",
		},
	)
	if err != nil {
		log.Error(err, "Unable to start manager")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		log.Error(err, "Unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		log.Error(err, "Unable to set up ready check")
		os.Exit(1)
	}
	// Add our own Agent Controller to the manager
	if err := controllers.Add(mgr); err != nil {
		log.Error(err, "Failure setting up Instana Agent Controller")
		os.Exit(1)
	}

	// controller-manager only runs controllers/runnables after getting the lock
	// we do the cleanup beforehand so our new deployment gets the lock
	log.Info("Deleting the controller-manager deployment and RBAC if it's present")
	//we need a new client because we have to delete old resources before starting the new manager
	if client, err := k8sClient.New(cfg, k8sClient.Options{
		Scheme: scheme,
	}); err != nil {
		log.Error(err, "Failed to create a new k8s client")
	} else {
		cleanupOldOperator(client)
	}

	log.Info("Starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Error(err, "Problem running manager")
		os.Exit(1)
	}
}

func cleanupOldOperator(client k8sClient.Client) {
	const InstanaNamespace string = "instana-agent"
	const InstanaOperatorOldDeploymentName string = "controller-manager"
	const InstanaOperatorOldClusterRoleName string = "manager-role"
	const InstanaOperatorOldClusterRoleBindingName string = "manager-rolebinding"

	log.Info("Delete the old deployment if present")
	oldDeployment := &appsv1.Deployment{}
	depKey := types.NamespacedName{
		Name:      InstanaOperatorOldDeploymentName,
		Namespace: InstanaNamespace,
	}
	if err := client.Get(context.Background(), depKey, oldDeployment); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Old operator deployment is not present in the cluster")
		} else {
			log.Error(err, "Failed to get the old operator deployment "+InstanaOperatorOldDeploymentName)
		}
	} else {
		log.Info("Deleting the old operator deployment " + InstanaOperatorOldDeploymentName)
		if err := client.Delete(context.Background(), oldDeployment); err != nil {
			log.Info("Failed to delete the old operator deployment " + InstanaOperatorOldDeploymentName)
		} else {
			log.Info("Successfully deleted the deployment " + InstanaOperatorOldDeploymentName)
		}
	}

	log.Info("Delete old RBAC resources if present")
	oldRole := &rbacv1.ClusterRole{}
	roleKey := types.NamespacedName{
		Name: InstanaOperatorOldClusterRoleName,
	}
	if err := client.Get(context.Background(), roleKey, oldRole); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Old operator clusterrole is not present in the cluster")
		} else {
			log.Error(err, "Failed to get the old operator clusterrole "+InstanaOperatorOldClusterRoleName)
		}
	} else {
		log.Info("Deleting the old operator clusterrole " + InstanaOperatorOldClusterRoleName)
		if err := client.Delete(context.Background(), oldRole); err != nil {
			log.Info("Failed to delete the old operator clusterrole " + InstanaOperatorOldClusterRoleName)
		} else {
			log.Info("Successfully deleted the clusterrole " + InstanaOperatorOldClusterRoleName)
		}
	}
	oldRoleBinding := &rbacv1.ClusterRoleBinding{}
	bindingKey := types.NamespacedName{
		Name: InstanaOperatorOldClusterRoleBindingName,
	}
	if err := client.Get(context.Background(), bindingKey, oldRoleBinding); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Old operator clusterrolebinding is not present in the cluster")
		} else {
			log.Error(err, "Failed to get the old operator clusterrolebinding "+InstanaOperatorOldClusterRoleBindingName)
		}
	} else {
		log.Info("Deleting the old operator clusterrolebinding " + InstanaOperatorOldClusterRoleBindingName)
		if err := client.Delete(context.Background(), oldRoleBinding); err != nil {
			log.Info("Failed to delete the old operator clusterrolebinding " + InstanaOperatorOldClusterRoleBindingName)
		} else {
			log.Info("Successfully deleted the clusterrolebinding " + InstanaOperatorOldClusterRoleBindingName)
		}
	}

}

func printVersion() {
	log.Info(fmt.Sprintf("Operator Version: %s", version.Version))
	log.Info(fmt.Sprintf("Operator Git Commit SHA: %s", version.GitCommit))
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
}
