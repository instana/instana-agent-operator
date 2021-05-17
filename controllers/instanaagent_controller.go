/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package controllers

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	instanaV1Beta1 "github.com/instana/instana-agent-operator/api/v1beta1"
	"github.com/pkg/errors"

	appV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"io"
	"log"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/output"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
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
	settings                = cli.New()
)

type InstanaAgentReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=agents.instana.com,namespace=instana-agent,resources=instanaagent,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=agents.instana.com,namespace=instana-agent,resources=instanaagent/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=agents.instana.com,namespace=instana-agent,resources=instanaagent/finalizers,verbs=update
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

	if err = upgradeInstallCharts(); err != nil {
		return ctrl.Result{}, err
	}
	log.Println("charts installed successfully")

	var reconcilationError = error(nil)

	// if err = r.reconcileSecrets(ctx, crdInstance); err != nil {
	// 	reconcilationError = err
	// }
	// if err = r.reconcileImagePullSecrets(ctx, crdInstance); err != nil {
	// 	reconcilationError = err
	// }
	// if err = r.reconcileServices(ctx, crdInstance); err != nil {
	// 	reconcilationError = err
	// }
	// if err = r.reconcileServiceAccounts(ctx, crdInstance); err != nil {
	// 	reconcilationError = err
	// }
	// if err = r.reconcileConfigMap(ctx, crdInstance); err != nil {
	// 	reconcilationError = err
	// }
	// if err = r.reconcileClusterRole(ctx); err != nil {
	// 	reconcilationError = err
	// }
	// if err = r.reconcileClusterRoleBinding(ctx); err != nil {
	// 	reconcilationError = err
	// }
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
		Owns(&coreV1.ConfigMap{}).
		Owns(&coreV1.Service{}).
		Owns(&coreV1.ServiceAccount{}).
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
func upgradeInstallCharts() error {
	cfg := new(action.Configuration)
	// You can pass an empty string instead of settings.Namespace() to list
	// all namespaces
	if err := cfg.Init(settings.RESTClientGetter(), settings.Namespace(), os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
		log.Printf("%+v", err)
		os.Exit(1)
	}
	client := action.NewUpgrade(cfg)
	var outfmt output.Format
	var createNamespace bool
	client.Namespace = settings.Namespace()
	client.Install = true
	valueOpts := &values.Options{Values: []string{
		"agent.key='2Zykc2m_RiKJnVE-TNNdrA'",
		"agent.endpointHost='ingress-pink-saas.instana.rocks'",
		"cluster.name='docker-desktop'",
		"zone.name='OhMyHelm'",
		"agent.logLevel='DEBUG'",
		"agent.image.name=gcr.io/instana-agent-qa/instana/agent/dev",
		"agent.image.tag=latest",
	}}
	args := []string{
		"instana-agent",
		"/Users/yousefabdelhamid/instana/instana-agent-charts/target/helm-charts/v1.99.9/",
	}

	if client.Install {
		// If a release does not exist, install it.
		histClient := action.NewHistory(cfg)
		histClient.Max = 1
		if _, err := histClient.Run(args[0]); err == driver.ErrReleaseNotFound {
			// Only print this to stdout for table output
			if outfmt == output.Table {
				fmt.Fprintf(os.Stdout, "Release %q does not exist. Installing it now.\n", args[0])
			}
			instClient := action.NewInstall(cfg)
			instClient.CreateNamespace = createNamespace
			instClient.ChartPathOptions = client.ChartPathOptions
			instClient.DryRun = client.DryRun
			instClient.DisableHooks = client.DisableHooks
			instClient.SkipCRDs = client.SkipCRDs
			instClient.Timeout = client.Timeout
			instClient.Wait = client.Wait
			instClient.WaitForJobs = client.WaitForJobs
			instClient.Devel = client.Devel
			instClient.Namespace = client.Namespace
			instClient.Atomic = client.Atomic
			instClient.PostRenderer = client.PostRenderer
			instClient.DisableOpenAPIValidation = client.DisableOpenAPIValidation
			instClient.SubNotes = client.SubNotes
			instClient.Description = client.Description

			rel, err := runInstall(args, instClient, valueOpts, os.Stdout)
			if err != nil {
				log.Println(err)
				return err
			}
			log.Println("done installing")
			log.Println(rel)
			return nil
		} else if err != nil {
			return err
		}
	}

	if client.Version == "" && client.Devel {
		// debug("setting version to >0.0.0-0")
		client.Version = ">0.0.0-0"
	}

	chartPath, err := client.ChartPathOptions.LocateChart(args[1], settings)
	if err != nil {
		return err
	}

	vals, err := valueOpts.MergeValues(getter.All(settings))
	if err != nil {
		return err
	}

	// Check chart dependencies to make sure all are present in /charts
	ch, err := loader.Load(chartPath)
	if err != nil {
		return err
	}
	if req := ch.Metadata.Dependencies; req != nil {
		if err := action.CheckDependencies(ch, req); err != nil {
			return err
		}
	}

	if ch.Metadata.Deprecated {
		log.Println("This chart is deprecated")
	}

	rel, err := client.Run(args[0], ch, vals)
	if err != nil {
		return errors.Wrap(err, "UPGRADE FAILED")
	}

	if outfmt == output.Table {
		fmt.Fprintf(os.Stdout, "Release %q has been upgraded. Happy Helming!\n", args[0])
	}
	log.Println("done upgrading")
	log.Println(rel)
	return nil

}
func installCharts() error {
	actionConfig := new(action.Configuration)
	// You can pass an empty string instead of settings.Namespace() to list
	// all namespaces
	if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
		log.Printf("%+v", err)
		os.Exit(1)
	}
	client := action.NewInstall(actionConfig)
	valueOpts := &values.Options{Values: []string{
		"agent.key='2Zykc2m_RiKJnVE-TNNdrA'",
		"agent.endpointHost='ingress-pink-saas.instana.rocks'",
		"cluster.name='docker-desktop'",
		"zone.name='OhMyHelm'",
		"agent.logLevel='DEBUG'",
		"agent.image.name=gcr.io/instana-agent-qa/instana/agent/dev",
		"agent.image.tag=latest",
	}}
	args := []string{
		"instana-agent",
		"/Users/yousefabdelhamid/instana/instana-agent-charts/target/helm-charts/v1.99.9/",
	}

	rel, err := runInstall(args, client, valueOpts, os.Stdout)
	if err != nil {
		log.Println(err)
		return err
	}
	log.Println("done installing")
	log.Println(rel)
	return nil
}

func runInstall(args []string, client *action.Install, valueOpts *values.Options, out io.Writer) (*release.Release, error) {
	client.Version = AppVersion

	_, chart, err := client.NameAndChart(args)
	if err != nil {
		return nil, err
	}
	client.ReleaseName = AppName

	cp, err := client.ChartPathOptions.LocateChart(chart, settings)
	if err != nil {
		return nil, err
	}

	log.Printf("CHART PATH: %s\n", cp)

	// Check chart dependencies to make sure all are present in /charts
	chartRequested, err := loader.Load(cp)
	if err != nil {
		return nil, err
	}

	p := getter.All(settings)

	vals, err := valueOpts.MergeValues(p)
	if err != nil {
		return nil, err
	}
	if err := checkIfInstallable(chartRequested); err != nil {
		return nil, err
	}

	if chartRequested.Metadata.Deprecated {
		log.Printf("This chart is deprecated")
	}

	if req := chartRequested.Metadata.Dependencies; req != nil {
		// If CheckDependencies returns an error, we have unfulfilled dependencies.
		// As of Helm 2.4.0, this is treated as a stopping condition:
		// https://github.com/helm/helm/issues/2209
		if err := action.CheckDependencies(chartRequested, req); err != nil {
			if client.DependencyUpdate {
				man := &downloader.Manager{
					Out:              out,
					ChartPath:        cp,
					Keyring:          client.ChartPathOptions.Keyring,
					SkipUpdate:       false,
					Getters:          p,
					RepositoryConfig: settings.RepositoryConfig,
					RepositoryCache:  settings.RepositoryCache,
					Debug:            settings.Debug,
				}
				if err := man.Update(); err != nil {
					return nil, err
				}
				// Reload the chart with the updated Chart.lock file.
				if chartRequested, err = loader.Load(cp); err != nil {
					return nil, errors.Wrap(err, "failed reloading chart after repo update")
				}
			} else {
				return nil, err
			}
		}
	}

	client.Namespace = "instana-agent"
	client.CreateNamespace = true
	// chartRequested.Metadata.Version = AppVersion
	return client.Run(chartRequested, vals)
}

// checkIfInstallable validates if a chart can be installed
//
// Application chart type is only installable
func checkIfInstallable(ch *chart.Chart) error {
	switch ch.Metadata.Type {
	case "", "application":
		return nil
	}
	return errors.Errorf("%s charts are not installable", ch.Metadata.Type)
}
