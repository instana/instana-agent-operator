/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/pflag"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"

	agentoperatorv1beta1 "github.com/instana/instana-agent-operator/api/v1beta1"
	"github.com/instana/instana-agent-operator/logger"
	"github.com/instana/instana-agent-operator/version"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	//+kubebuilder:scaffold:imports
)

var (
	scheme = k8sruntime.NewScheme()
	log    = logger.NewAgentLogger()
)

//var subcmdCallbacks = map[string]func(ns string, cfg *rest.Config) (manager.Manager, error){
//	//"operator": startOperator,
//}
//
//var errBadSubcmd = errors.New("subcommand must be operator")
//
//var (
//	certsDir string
//	certFile string
//	keyFile  string
//)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(agentoperatorv1beta1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

//var globalLog = logf.Log.WithName("global")

func main() {
	pflag.Parse()

	ctrl.SetLogger(logger.NewAgentLogger())

	printVersion()

	// subcmd := "operator"
	// if args := pflag.Args(); len(args) > 0 {
	// 	subcmd = args[0]
	// }

	// subcmdFn := subcmdCallbacks[subcmd]
	// if subcmdFn == nil {
	// 	log.Error(errBadSubcmd, "Unknown subcommand", "command", subcmd)
	// 	os.Exit(1)
	// }

	// namespace := os.Getenv("POD_NAMESPACE")

	// cfg, err := config.GetConfig()

	// //var enableLeaderElection bool
	// //var probeAddr string
	// //flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	// //flag.BoolVar(&enableLeaderElection, "leader-elect", false,
	// //  "Enable leader election for controller manager. "+
	// //      "Enabling this will ensure there is only one active controller manager.")
	// //opts := zap.Options{
	// //  Development: true,
	// //}
	// //opts.BindFlags(flag.CommandLine)
	// //flag.Parse()
	// //
	// //ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	// //
	// //mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
	// //  Scheme:                 scheme,
	// //  MetricsBindAddress:     metricsAddr,
	// //  Port:                   9443,
	// //  HealthProbeBindAddress: probeAddr,
	// //  LeaderElection:         enableLeaderElection,
	// //  LeaderElectionID:       "819a9291.instana.com",
	// //})

	// if err != nil {
	// 	log.Error(err, "")
	// 	os.Exit(1)
	// }

	// mgr, err := subcmdFn(namespace, cfg)
	// if err != nil {
	// 	log.Error(err, "")
	// 	os.Exit(1)
	// }

	// signalHandler := ctrl.SetupSignalHandler()

	// //if err = (&controllers.PodSetReconciler{
	// //  Client: mgr.GetClient(),
	// //  Log:    ctrl.Log.WithName("controllers").WithName("PodSet"),
	// //  Scheme: mgr.GetScheme(),
	// //}).SetupWithManager(mgr); err != nil {
	// //  setupLog.Error(err, "unable to create controller", "controller", "PodSet")
	// //  os.Exit(1)
	// //}
	// ////+kubebuilder:scaffold:builder
	// //
	// //if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
	// //  setupLog.Error(err, "unable to set up health check")
	// //  os.Exit(1)
	// //}
	// //if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
	// //  setupLog.Error(err, "unable to set up ready check")
	// //  os.Exit(1)
	// //}

	// log.Info("starting manager")
	// if err := mgr.Start(signalHandler); err != nil {
	// 	log.Error(err, "problem running manager")
	// 	os.Exit(1)
	// }

}

func printVersion() {
	log.Info(fmt.Sprintf("Operator Version: %s", version.Version))
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
}
