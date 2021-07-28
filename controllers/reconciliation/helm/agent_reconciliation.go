/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package helm

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-logr/logr"
	instanaV1Beta1 "github.com/instana/instana-agent-operator/api/v1beta1"
	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	"helm.sh/helm/v3/pkg/storage/driver"
	"sigs.k8s.io/yaml"
)

const (
	helm_repo = "https://agents.instana.io/helm"
)

var (
	AppName        = "instana-agent"
	AgentNameSpace = AppName

	settings = cli.New()
	HelmCfg  = new(action.Configuration)
)

type HelmReconciliation struct {
	Client client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

func (h *HelmReconciliation) DebugLog(format string, v ...interface{}) {
	h.Log.WithName("helm").V(1).Info(fmt.Sprintf(format, v...))
}

func (h *HelmReconciliation) Delete(crdInstance *instanaV1Beta1.InstanaAgent) error {
	settings.RepositoryConfig = helm_repo
	if err := HelmCfg.Init(settings.RESTClientGetter(), settings.Namespace(), os.Getenv("HELM_DRIVER"), h.DebugLog); err != nil {
		return err
	}

	uninstallAction := action.NewUninstall(HelmCfg)
	_, err := uninstallAction.Run(AppName)
	if err != nil {
		// If there was an error because already uninstalled, ignore it
		// Unfortunately the Helm library doesn't return nice Error types so can only check on message
		if strings.Contains(err.Error(), "uninstall: Release not loaded") {
			h.Log.Info("Ignoring error during Instana Agent deletion, Helm resources already removed")
		} else {
			return err
		}
	}

	h.Log.Info("Release uninstalled")
	return nil
}

func (h *HelmReconciliation) CreateOrUpdate(req ctrl.Request, crdInstance *instanaV1Beta1.InstanaAgent) error {
	settings.RepositoryConfig = helm_repo
	if err := HelmCfg.Init(settings.RESTClientGetter(), settings.Namespace(), os.Getenv("HELM_DRIVER"), h.DebugLog); err != nil {
		return err
	}
	client := action.NewUpgrade(HelmCfg)
	var createNamespace bool
	client.Namespace = settings.Namespace()
	client.Install = true
	client.RepoURL = helm_repo
	client.PostRenderer = &AgentPostRenderer{
		Scheme:      h.Scheme,
		CrdInstance: crdInstance,
		Client:      h.Client,
		HelmCfg:     *HelmCfg,
	}
	client.MaxHistory = 1
	args := []string{AppName, AppName}

	versionSet, err := h.getApiVersions()
	if err != nil {
		return err
	}
	if versionSet.Has("apps.openshift.io/v1") {
		crdInstance.Spec.OpenShift = true
	} else {
		crdInstance.Spec.OpenShift = false
	}

	yamlMap, err := h.mapCRDToYaml(crdInstance)
	if err != nil {
		return err
	}

	if client.Install {
		// If a release does not exist, install it.
		histClient := action.NewHistory(HelmCfg)
		histClient.Max = 1
		if _, err := histClient.Run(args[0]); err == driver.ErrReleaseNotFound {
			h.Log.Info("Release does not exist. Installing it now.")
			instClient := action.NewInstall(HelmCfg)
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
			_, err := h.runInstall(args, instClient, yamlMap)
			if err != nil {
				return err
			}
			h.Log.Info("done installing")
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
		h.Log.Info("This chart is deprecated")
	}

	_, err = client.Run(args[0], ch, yamlMap)
	if err != nil {
		return errors.Wrap(err, "UPGRADE FAILED")
	}

	h.Log.Info("done upgrading")
	return nil

}

func (h *HelmReconciliation) repoUpdate() error {
	entry := &repo.Entry{Name: AppName, URL: helm_repo}
	r, err := repo.NewChartRepository(entry, getter.All(settings))
	if err != nil {
		return err
	}
	if _, err := r.DownloadIndexFile(); err != nil {
		h.Log.Info(fmt.Sprintf("...Unable to get an update from the %q chart repository (%s):\n\t%s\n", r.Config.Name, r.Config.URL, err))
	} else {
		h.Log.Info(fmt.Sprintf("...Successfully got an update from the %q chart repository\n", r.Config.Name))
	}
	return nil
}
func (h *HelmReconciliation) runInstall(args []string, client *action.Install, yamlMap map[string]interface{}) (*release.Release, error) {

	_, chart, err := client.NameAndChart(args)
	if err != nil {
		return nil, err
	}
	client.ReleaseName = AppName

	cp, err := client.ChartPathOptions.LocateChart(chart, settings)
	if err != nil {
		if err = h.repoUpdate(); err != nil {
			return nil, err
		}
	}
	// Check chart dependencies to make sure all are present in /charts
	chartRequested, err := loader.Load(cp)
	if err != nil {
		return nil, err
	}

	p := getter.All(settings)
	if err := h.checkIfInstallable(chartRequested); err != nil {
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

	client.Namespace = AgentNameSpace
	client.CreateNamespace = true
	return client.Run(chartRequested, yamlMap)
}

func (h *HelmReconciliation) checkIfInstallable(ch *chart.Chart) error {
	switch ch.Metadata.Type {
	case "", "application":
		return nil
	}
	return errors.Errorf("%s charts are not installable", ch.Metadata.Type)
}

func (h *HelmReconciliation) mapCRDToYaml(crdInstance *instanaV1Beta1.InstanaAgent) (map[string]interface{}, error) {
	specYaml, err := yaml.Marshal(crdInstance.Spec)
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshaling to yaml")
	}
	yamlMap := map[string]interface{}{}
	if err := yaml.Unmarshal(specYaml, &yamlMap); err != nil {
		return nil, err
	}
	return yamlMap, nil
}
func (h *HelmReconciliation) getApiVersions() (*chartutil.VersionSet, error) {
	if HelmCfg.Capabilities != nil {
		return &HelmCfg.Capabilities.APIVersions, nil
	}
	dc, err := HelmCfg.RESTClientGetter.ToDiscoveryClient()
	if err != nil {
		return nil, errors.Wrap(err, "could not get Kubernetes discovery client")
	}
	// force a discovery cache invalidation to always fetch the latest server version/capabilities.
	dc.Invalidate()
	// Issue #6361:
	// Client-Go emits an error when an API service is registered but unimplemented.
	// We trap that error here and print a warning. But since the discovery client continues
	// building the API object, it is correctly populated with all valid APIs.
	// See https://github.com/kubernetes/kubernetes/issues/72051#issuecomment-521157642
	apiVersions, err := action.GetVersionSet(dc)
	if err != nil {
		if discovery.IsGroupDiscoveryFailedError(err) {
			h.Log.Error(err, "WARNING: The Kubernetes server has an orphaned API service. Server reports: %s")
			h.Log.Info("WARNING: To fix this, kubectl delete apiservice <service-name>")
		} else {
			return nil, errors.Wrap(err, "could not get apiVersions from Kubernetes")
		}
	}
	return &apiVersions, err
}
