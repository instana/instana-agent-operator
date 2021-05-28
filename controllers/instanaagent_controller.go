/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package controllers

import (
	"bytes"
	"context"
	"os"

	"github.com/go-logr/logr"
	instanaV1Beta1 "github.com/instana/instana-agent-operator/api/v1beta1"
	"github.com/pkg/errors"

	appV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/resource"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"log"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
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
	AgentImagePullSecretName = "containers-instana-io"
	DockerRegistry           = "containers.instana.io"

	AgentPort         = 42699
	OpenTelemetryPort = 55680
	helm_repo         = "https://agents.instana.io/helm"
)

var (
	AppName                 = "instana-agent"
	AgentNameSpace          = AppName
	AgentSecretName         = AppName
	AgentServiceAccountName = AppName
	settings                = cli.New()
	helmCfg                 = new(action.Configuration)
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

	specYaml, err := yaml.Marshal(crdInstance.Spec)
	if err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed marshaling to yaml")
	}
	yamlMap := map[string]interface{}{}
	if err := yaml.Unmarshal(specYaml, &yamlMap); err != nil {
		return ctrl.Result{}, err
	}
	log.Println(yamlMap)

	if err = r.upgradeInstallCharts(ctx, req, crdInstance, yamlMap); err != nil {
		return ctrl.Result{}, err
	}
	r.Log.Info("Charts installed/upgraded successfully")
	return ctrl.Result{}, nil
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

func (r *InstanaAgentReconciler) Run(in *bytes.Buffer) (*bytes.Buffer, error) {
	resourceList, err := helmCfg.KubeClient.Build(in, false)
	if err != nil {
		return nil, err
	}
	out := bytes.Buffer{}
	log.Println("rendered manifests")
	err = resourceList.Visit(func(r *resource.Info, err error) error {

		if err != nil {
			return err
		}

		objMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(r.Object)
		if err != nil {
			return err
		}
		if r.ObjectName() == "daemonsets/instana-agent" {
			var ds = &appV1.DaemonSet{}
			runtime.DefaultUnstructuredConverter.FromUnstructured(objMap, ds)
			containerList := ds.Spec.Template.Spec.Containers
			for i, container := range containerList {
				if container.Name == "leader-elector" {
					containerList = append(containerList[:i], containerList[i+1:]...)
					break
				}
			}
			ds.Spec.Template.Spec.Containers = containerList
			outData, err := yaml.Marshal(ds)
			if err != nil {
				return err
			}
			if _, err := out.WriteString("---\n" + string(outData)); err != nil {
				return err
			}
		} else {
			u := &unstructured.Unstructured{Object: objMap}

			outData, err := yaml.Marshal(u.Object)
			if err != nil {
				return err
			}
			if _, err := out.WriteString("---\n" + string(outData)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *InstanaAgentReconciler) upgradeInstallCharts(ctx context.Context, req ctrl.Request, crdInstance *instanaV1Beta1.InstanaAgent, yamlMap map[string]interface{}) error {
	// cfg := new(action.Configuration)
	settings.RepositoryConfig = helm_repo
	if err := helmCfg.Init(settings.RESTClientGetter(), settings.Namespace(), os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
		return err
	}
	client := action.NewUpgrade(helmCfg)
	var createNamespace bool
	client.Namespace = settings.Namespace()
	client.Install = true
	client.RepoURL = helm_repo
	client.PostRenderer = r

	args := []string{AppName, AppName}

	if client.Install {
		// If a release does not exist, install it.
		histClient := action.NewHistory(helmCfg)
		histClient.Max = 1
		if _, err := histClient.Run(args[0]); err == driver.ErrReleaseNotFound {
			r.Log.Info("Release does not exist. Installing it now.")
			instClient := action.NewInstall(helmCfg)
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
			instClient.RepoURL = helm_repo
			_, err := runInstall(args, instClient, yamlMap)
			if err != nil {
				return err
			}
			if err = r.setReferences(ctx, req, crdInstance); err != nil {
				return err
			}
			r.Log.Info("done installing")
			return nil
		} else if err != nil {
			return err
		}
	}

	if client.Version == "" && client.Devel {
		client.Version = ">0.0.0-0"
	}

	chartPath, err := client.ChartPathOptions.LocateChart(args[1], settings)
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
		r.Log.Info("This chart is deprecated")
	}

	_, err = client.Run(args[0], ch, yamlMap)
	if err != nil {
		return errors.Wrap(err, "UPGRADE FAILED")
	}

	r.Log.Info("done upgrading")
	return nil

}
func (r *InstanaAgentReconciler) setReferences(ctx context.Context, req ctrl.Request, crdInstance *instanaV1Beta1.InstanaAgent) error {
	var reconcilationError = error(nil)
	var err error
	if err = r.setSecretsReference(ctx, crdInstance); err != nil {
		reconcilationError = err
	}
	if err = r.setImagePullSecretsReference(ctx, crdInstance); err != nil {
		reconcilationError = err
	}
	if err = r.setServicesReference(ctx, crdInstance); err != nil {
		reconcilationError = err
	}
	if err = r.setServiceAccountsReference(ctx, crdInstance); err != nil {
		reconcilationError = err
	}
	if err = r.setConfigMapReference(ctx, crdInstance); err != nil {
		reconcilationError = err
	}
	if err = r.setDaemonsetReference(ctx, req, crdInstance); err != nil {
		reconcilationError = err
	}
	return reconcilationError
}

func runInstall(args []string, client *action.Install, yamlMap map[string]interface{}) (*release.Release, error) {

	_, chart, err := client.NameAndChart(args)
	if err != nil {
		return nil, err
	}
	client.ReleaseName = AppName

	cp, err := client.ChartPathOptions.LocateChart(chart, settings)
	if err != nil {
		return nil, err
	}

	// Check chart dependencies to make sure all are present in /charts
	chartRequested, err := loader.Load(cp)
	if err != nil {
		return nil, err
	}

	p := getter.All(settings)

	if err := checkIfInstallable(chartRequested); err != nil {
		return nil, err
	}

	if req := chartRequested.Metadata.Dependencies; req != nil {
		if err := action.CheckDependencies(chartRequested, req); err != nil {
			if client.DependencyUpdate {
				man := &downloader.Manager{
					Out:              os.Stdout,
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
	return client.Run(chartRequested, yamlMap)
}

func checkIfInstallable(ch *chart.Chart) error {
	switch ch.Metadata.Type {
	case "", "application":
		return nil
	}
	return errors.Errorf("%s charts are not installable", ch.Metadata.Type)
}
