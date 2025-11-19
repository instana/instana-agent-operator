/*
 * (c) Copyright IBM Corp. 2025
 */

package e2e

import (
	"context"
	stderrs "errors"
	"fmt"
	"sync"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

var (
	// ErrFullResetRequired signals that the fast cleanup path cannot proceed and the legacy full reset is required.
	ErrFullResetRequired = stderrs.New("full reset required")
	// ErrOperatorDeploymentNotFound is returned when the operator deployment cannot be located.
	ErrOperatorDeploymentNotFound = stderrs.New("operator deployment not found")
)

type suiteState struct {
	mu                 sync.Mutex
	fullResetRequested bool
	fullResetReason    string
}

var currentSuiteState = suiteState{}

// MarkFullResetRequired allows tests to mark the environment as dirty, forcing the next test to perform a full cleanup.
func MarkFullResetRequired(reason string) {
	currentSuiteState.mu.Lock()
	defer currentSuiteState.mu.Unlock()
	currentSuiteState.fullResetRequested = true
	currentSuiteState.fullResetReason = reason
}

// FullResetRequested returns whether a full cleanup has been requested together with the reason.
func FullResetRequested() (bool, string) {
	currentSuiteState.mu.Lock()
	defer currentSuiteState.mu.Unlock()
	return currentSuiteState.fullResetRequested, currentSuiteState.fullResetReason
}

// ClearFullResetRequest clears a previously recorded full cleanup request.
func ClearFullResetRequest() {
	currentSuiteState.mu.Lock()
	defer currentSuiteState.mu.Unlock()
	currentSuiteState.fullResetRequested = false
	currentSuiteState.fullResetReason = ""
}

func desiredDevBuildImage() string {
	return fmt.Sprintf("%s:%s", InstanaTestCfg.OperatorImage.Name, InstanaTestCfg.OperatorImage.Tag)
}

func getOperatorDeployment(ctx context.Context, cfg *envconf.Config) (*appsv1.Deployment, error) {
	r, err := resources.New(cfg.Client().RESTConfig())
	if err != nil {
		return nil, fmt.Errorf("initializing client for operator deployment lookup: %w", err)
	}
	r.WithNamespace(cfg.Namespace())
	deployment := &appsv1.Deployment{}
	err = r.Get(ctx, InstanaOperatorDeploymentName, cfg.Namespace(), deployment)
	if apierrors.IsNotFound(err) {
		return nil, ErrOperatorDeploymentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("fetching operator deployment: %w", err)
	}
	return deployment, nil
}

func currentOperatorImage(ctx context.Context, cfg *envconf.Config) (string, error) {
	deployment, err := getOperatorDeployment(ctx, cfg)
	if err != nil {
		return "", err
	}
	containers := deployment.Spec.Template.Spec.Containers
	if len(containers) == 0 {
		return "", fmt.Errorf("operator deployment does not define containers")
	}
	return containers[0].Image, nil
}
