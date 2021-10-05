/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package v1

import (
	ctrl "sigs.k8s.io/controller-runtime"
)

// log is for logging in this package.
//var instanaagentlog = logf.Log.WithName("instanaagent-resource")

func (r *InstanaAgent) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// NOTE: The below is for a "MutatingAdmissionWebhook", not to be confused with a "ConvertingWebhook" to just convert CRD versions
//+kubebuilder:webhook:path=/mutate-agent-spec-v1-instanaagent,mutating=true,failurePolicy=fail,sideEffects=None,groups=instana.io,resources=agents,verbs=create;update,versions=v1,name=magents.kb.io,admissionReviewVersions={v1,v1beta1}

//var _ webhook.Defaulter = &InstanaAgent{}
//
//// Default implements webhook.Defaulter so a webhook will be registered for the type
//func (r *InstanaAgent) Default() {
//	instanaagentlog.Info("default", "name", r.Name)
//}
