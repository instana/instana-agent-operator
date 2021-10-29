/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"

	"github.com/instana/instana-agent-operator/controllers"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	k8sruntime "k8s.io/apimachinery/pkg/runtime"

	agentoperatorv1 "github.com/instana/instana-agent-operator/api/v1"
	agentoperatorv1beta1 "github.com/instana/instana-agent-operator/api/v1beta1"
	"github.com/instana/instana-agent-operator/version"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
)

var (
	scheme = k8sruntime.NewScheme()
	log    = logf.Log.WithName("main")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(agentoperatorv1beta1.AddToScheme(scheme))
	utilruntime.Must(agentoperatorv1.AddToScheme(scheme))
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
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	// When running in debug-mode, include some more logging etc
	debugMode, _ := strconv.ParseBool(os.Getenv("DEBUG_MODE"))
	// For local dev mode, make it possible to override the Certificate Path where certificates might get stored
	certificatePath := os.Getenv("CERTIFICATE_PATH")

	opts := zap.Options{
		Development: debugMode,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	// Set the Logger to be used also by the controller-runtime
	logf.SetLogger(zap.New(zap.UseFlagOptions(&opts)).WithName("instana"))

	printVersion()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Namespace:              "instana-agent",
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "819a9291.instana.io",
		CertDir:                certificatePath,
	})
	if err != nil {
		log.Error(err, "Unable to start manager")
		os.Exit(1)
	}

	// Register our Conversion Webhook to translate v1beta1 to v1 versions
	// WebHooks are by default enabled, so "empty" variable should not be interpreted as "false"
	if enableWebhooks, err := strconv.ParseBool(os.Getenv("ENABLE_WEBHOOKS")); enableWebhooks || err != nil {
		if err = (&agentoperatorv1.InstanaAgent{}).SetupWebhookWithManager(mgr); err != nil {
			log.Error(err, "unable to create webhook", "webhook", "InstanaAgent")
			os.Exit(1)
		}
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

	log.Info("Starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Error(err, "Problem running manager")
		os.Exit(1)
	}

}

func printVersion() {
	log.Info(fmt.Sprintf("Operator Version: %s", version.Version))
	log.Info(fmt.Sprintf("Operator Git Commit SHA: %s", version.GitCommit))
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
}
