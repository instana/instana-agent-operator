/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package helm

import (
	"fmt"
	"os"

	"sigs.k8s.io/yaml"

	"github.com/go-logr/logr"
	instanaV1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
	"helm.sh/helm/v3/pkg/storage/driver"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	helmRepo       = "https://agents.instana.io/helm"
	agentChartName = "instana-agent"
)

var (
	settings *cli.EnvSettings
)

func init() {
	settings = cli.New()
	settings.RepositoryConfig = helmRepo
}

func NewHelmReconciliation(scheme *runtime.Scheme, log logr.Logger, crAppName string, crAppNamespace string) *HelmReconciliation {
	h := &HelmReconciliation{
		scheme:         scheme,
		log:            log.WithName("helm-reconcile"),
		crAppName:      crAppName,
		crAppNamespace: crAppNamespace,
	}
	if err := h.initHelmConfig(); err != nil {
		// This is a highly unlikely edge-case (the action.Configuration.Init(...) itself only panics) from which we can't
		// continue.
		h.log.Error(err, "Failed initializing Helm Reconciliation")
		panic(fmt.Sprintf("Failed initializing Helm Reconciliation: %v", err))
	}

	return h
}

// initHelmConfig initializes some general setup, extracted from the 'constructor' because it might error
func (h *HelmReconciliation) initHelmConfig() error {
	h.helmCfg = new(action.Configuration)
	if err := h.helmCfg.Init(settings.RESTClientGetter(), settings.Namespace(), os.Getenv("HELM_DRIVER"), h.debugLog); err != nil {
		return fmt.Errorf("failure initializing Helm configuration: %w", err)
	}
	return nil
}

// debugLog provides a logging function so that the Helm driver will send all output via our logging pipeline
func (h *HelmReconciliation) debugLog(format string, v ...interface{}) {
	h.log.WithName("helm").V(1).Info(fmt.Sprintf(format, v...))
}

type HelmReconciliation struct {
	helmCfg        *action.Configuration
	scheme         *runtime.Scheme
	log            logr.Logger
	crAppName      string
	crAppNamespace string
}

func (h *HelmReconciliation) Delete(_ ctrl.Request, _ *instanaV1.InstanaAgent) error {
	uninstallAction := action.NewUninstall(h.helmCfg)
	if _, err := uninstallAction.Run(h.crAppName); err != nil {
		// If there was an error because already uninstalled, ignore it
		if errors.Is(err, driver.ErrReleaseNotFound) {
			h.log.Info("Ignoring error during Instana Agent deletion, Helm resources already removed")
			h.log.V(1).Info("Helm Uninstall error", "error", err)
		} else {
			return err
		}
	}

	h.log.Info("Release Instana Agent (Charts) uninstalled")
	return nil
}

func (h *HelmReconciliation) CreateOrUpdate(_ ctrl.Request, crdInstance *instanaV1.InstanaAgent) error {
	// Prepare CRD for Helm Chart installation, converting to YAML (key-value map)
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

	// Find out if there's an Agent chart already installed or need a fresh install
	histClient := action.NewHistory(h.helmCfg)
	histClient.Max = 1
	if _, err := histClient.Run(h.crAppName); err == driver.ErrReleaseNotFound {
		h.log.Info("Instana Agent Chart Release does not exist. Installing it now.")

		installAction := action.NewInstall(h.helmCfg)
		installAction.CreateNamespace = true // Namespace should already be there (same as Operator), but won't hurt
		installAction.Namespace = h.crAppNamespace
		installAction.ReleaseName = h.crAppName
		installAction.RepoURL = helmRepo
		installAction.PostRenderer = NewAgentChartPostRenderer(h, crdInstance)
		installAction.Version = fixChartVersion(crdInstance.Spec.PinnedChartVersion, installAction.Devel)

		agentChart, err := h.loadAndValidateChart(installAction.ChartPathOptions)
		if err != nil {
			h.log.Error(err, "Failure loading or validating Instana Agent Helm Chart, cannot proceed installation")
			return errors.Wrap(err, "loading Instana Agent Helm Chart failed")
		}

		_, err = installAction.Run(agentChart, yamlMap)
		if err != nil {
			h.log.Error(err, "Failure installing Instana Agent Helm Chart, cannot proceed installation")
			return errors.Wrap(err, "installation Instana Agent failed")
		}

	} else if err == nil {
		h.log.Info("Found existing Instana Agent Chart Release. Upgrade existing installation.")

		upgradeAction := action.NewUpgrade(h.helmCfg)
		upgradeAction.Namespace = h.crAppNamespace
		upgradeAction.RepoURL = helmRepo
		upgradeAction.MaxHistory = 1
		upgradeAction.PostRenderer = NewAgentChartPostRenderer(h, crdInstance)
		upgradeAction.Version = fixChartVersion(crdInstance.Spec.PinnedChartVersion, upgradeAction.Devel)

		agentChart, err := h.loadAndValidateChart(upgradeAction.ChartPathOptions)
		if err != nil {
			h.log.Error(err, "Failure loading or validating Instana Agent Helm Chart, cannot proceed installation")
			return errors.Wrap(err, "loading Instana Agent Helm Chart failed")
		}

		_, err = upgradeAction.Run(h.crAppName, agentChart, yamlMap)
		if err != nil {
			h.log.Error(err, "Failure installing Instana Agent Helm Chart, cannot proceed installation")
			return errors.Wrap(err, "installation Instana Agent failed")
		}

	} else {
		h.log.Error(err, "Unexpected error trying to fetch Instana Agent Chart install history")
		return err
	}

	h.log.Info("Done installing / upgrading Instana Agent Helm Chart")
	return nil
}

func (h *HelmReconciliation) mapCRDToYaml(crdInstance *instanaV1.InstanaAgent) (map[string]interface{}, error) {
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

func fixChartVersion(version string, devel bool) string {
	if (len(version) == 0 || version == "") && devel {
		return ">0.0.0-0"
	} else {
		return version
	}
}

func (h *HelmReconciliation) loadAndValidateChart(chartOptions action.ChartPathOptions) (*chart.Chart, error) {
	chartPath, err := chartOptions.LocateChart(agentChartName, settings)
	if err != nil {
		// The Chart might have never been downloaded or got removed. Update the repo and fetch Chart locally
		if err := h.repoUpdate(); err != nil {
			return nil, err
		}
	}

	agentChart, err := loader.Load(chartPath)
	if err != nil {
		return nil, err
	}

	if agentChart.Metadata.Deprecated {
		h.log.Info("NOTE! This chart is deprecated!")
	}

	if err := h.checkIfInstallable(agentChart); err != nil {
		return nil, err
	}

	// NOTE, the 'original' code from the cmd/helm/install.go package contained code to also check and update Chart Dependencies.
	// Our Agent Chart does not have any dependencies on other Charts, so for simplicity this code is not needed and omitted for now.

	return agentChart, nil
}

func (h *HelmReconciliation) repoUpdate() error {
	entry := &repo.Entry{Name: agentChartName, URL: helmRepo}
	r, err := repo.NewChartRepository(entry, getter.All(settings))
	if err != nil {
		return err
	}
	if _, err := r.DownloadIndexFile(); err != nil {
		h.log.Info(fmt.Sprintf("...Unable to get an update from the %q chart repository (%s):\n\t%s\n", r.Config.Name, r.Config.URL, err))
	} else {
		h.log.Info(fmt.Sprintf("...Successfully got an update from the %q chart repository\n", r.Config.Name))
	}
	return nil
}

func (h *HelmReconciliation) checkIfInstallable(ch *chart.Chart) error {
	switch ch.Metadata.Type {
	case "", "application":
		return nil
	}
	return errors.Errorf("%s charts are not installable", ch.Metadata.Type)
}

func (h *HelmReconciliation) getApiVersions() (*chartutil.VersionSet, error) {
	if h.helmCfg.Capabilities != nil {
		return &h.helmCfg.Capabilities.APIVersions, nil
	}
	dc, err := h.helmCfg.RESTClientGetter.ToDiscoveryClient()
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
			h.log.Error(err, "WARNING: The Kubernetes server has an orphaned API service. Server reports: %s")
			h.log.Info("WARNING: To fix this, kubectl delete apiservice <service-name>")
		} else {
			return nil, errors.Wrap(err, "could not get apiVersions from Kubernetes")
		}
	}
	return &apiVersions, err
}
