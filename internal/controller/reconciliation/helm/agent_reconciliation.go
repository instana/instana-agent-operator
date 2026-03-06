/*
 * (c) Copyright IBM Corp. 2021, 2026
 * (c) Copyright Instana Inc. 2021, 2026
 */

package helm

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/cli"
	"helm.sh/helm/v4/pkg/storage/driver"
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
	); err != nil {
		return fmt.Errorf("failure initializing Helm configuration: %w", err)
	}
	// In Helm v4, logging is configured via SetLogger with an slog.Handler
	h.helmCfg.SetLogger(h.newSlogHandler())
	return nil
}

// newSlogHandler creates an slog.Handler that bridges to our logr.Logger
func (h *helmReconciliation) newSlogHandler() slog.Handler {
	return slog.NewTextHandler(
		&logrWriter{log: h.log.WithName("helm").V(1)},
		&slog.HandlerOptions{Level: slog.LevelDebug},
	)
}

// logrWriter adapts logr.Logger to io.Writer for slog
type logrWriter struct {
	log logr.Logger
}

func (w *logrWriter) Write(p []byte) (n int, err error) {
	w.log.Info(string(p))
	return len(p), nil
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
