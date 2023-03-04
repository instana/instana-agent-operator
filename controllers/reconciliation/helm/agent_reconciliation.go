/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package helm

import (
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/apimachinery/pkg/runtime"
)

type DeprecatedInternalChartUninstaller interface {
	Delete() error
}

func NewHelmReconciliation(
	scheme *runtime.Scheme,
	log logr.Logger,
) DeprecatedInternalChartUninstaller {
	h := &helmReconciliation{
		scheme:         scheme,
		log:            log.WithName("helm-reconcile"),
		crAppName:      "instana-agent",
		crAppNamespace: "instana-agent",
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
func (h *helmReconciliation) initHelmConfig() error {
	settings := cli.New()
	h.helmCfg = new(action.Configuration)
	if err := h.helmCfg.Init(
		settings.RESTClientGetter(),
		settings.Namespace(),
		os.Getenv("HELM_DRIVER"),
		h.debugLog,
	); err != nil {
		return fmt.Errorf("failure initializing Helm configuration: %w", err)
	}
	return nil
}

// debugLog provides a logging function so that the Helm driver will send all output via our logging pipeline
func (h *helmReconciliation) debugLog(format string, v ...interface{}) {
	h.log.WithName("helm").V(1).Info(fmt.Sprintf(format, v...))
}

type helmReconciliation struct {
	helmCfg        *action.Configuration
	scheme         *runtime.Scheme
	log            logr.Logger
	crAppName      string
	crAppNamespace string
}

func (h *helmReconciliation) Delete() error {
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
