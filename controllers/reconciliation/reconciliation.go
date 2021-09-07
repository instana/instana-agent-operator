/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package reconciliation

import (
	"github.com/go-logr/logr"
	instanaV1Beta1 "github.com/instana/instana-agent-operator/api/v1beta1"
	"github.com/instana/instana-agent-operator/controllers/reconciliation/helm"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Reconciliation interface {
	// CreateOrUpdate creates a new Agent installation or updates to the latest defined configuration
	CreateOrUpdate(req ctrl.Request, crdInstance *instanaV1Beta1.InstanaAgent) error
	// Delete removes the Agent installation from the cluster
	Delete(req ctrl.Request, crdInstance *instanaV1Beta1.InstanaAgent) error
}

func New(scheme *runtime.Scheme, log logr.Logger, crAppName string, crAppNamespace string) Reconciliation {
	return helm.NewHelmReconciliation(scheme, log, crAppName, crAppNamespace)
}
