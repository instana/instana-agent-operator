/*
(c) Copyright IBM Corp. 2025
(c) Copyright Instana Inc. 2025
*/

package certs

import (
	admissionv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

type webhookConfigBuilder struct {
	*instanav1.InstanaAgent
	helpers     helpers.Helpers
	isOpenShift bool
	caCertPem   []byte
}

func (wc *webhookConfigBuilder) IsNamespaced() bool {
	return false
}

func (wc *webhookConfigBuilder) ComponentName() string {
	return wc.helpers.AutotraceWebhookResourcesName()
}

func (wc *webhookConfigBuilder) getOCPAnnotions() map[string]string {
	var annotations map[string]string
	if wc.isOpenShift {
		annotations = map[string]string{
			"service.beta.openshift.io/inject-cabundle": "true",
		}
	}
	return annotations
}

func (wc *webhookConfigBuilder) getLabels() map[string]string {
	var labels map[string]string
	if wc.isOpenShift {
		labels = map[string]string{
			"autotrace": "instana-autotrace-webhook-impl",
		}
	}
	return labels
}

func (wc *webhookConfigBuilder) Build() (res optional.Optional[client.Object]) {

	failurePolicy := admissionv1.Fail //TODO: revert to Ignore
	reinvocationPolicy := admissionv1.IfNeededReinvocationPolicy
	matchPolicy := admissionv1.Equivalent
	sideEffect := admissionv1.SideEffectClassNoneOnDryRun

	return optional.Of[client.Object](
		&admissionv1.MutatingWebhookConfiguration{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "admissionregistration.k8s.io/v1",
				Kind:       "MutatingWebhookConfiguration",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        wc.ComponentName(),
				Labels:      wc.getLabels(),
				Annotations: wc.getOCPAnnotions(),
			},
			Webhooks: []admissionv1.MutatingWebhook{
				{
					Name:               "autotrace-webhook.instana.com",
					FailurePolicy:      &failurePolicy,
					ReinvocationPolicy: &reinvocationPolicy,
					MatchPolicy:        &matchPolicy,
					Rules: []admissionv1.RuleWithOperations{
						{
							Rule: admissionv1.Rule{
								APIGroups:   []string{""},
								APIVersions: []string{"v1", "v1beta1"},
								Resources:   []string{"pods", "configmaps"},
							},
							Operations: []admissionv1.OperationType{
								admissionv1.Create,
								admissionv1.Update,
							},
						},
						{
							Rule: admissionv1.Rule{
								APIGroups:   []string{"apps"},
								APIVersions: []string{"v1", "v1beta1"},
								Resources:   []string{"deployments", "daemonsets", "replicaset", "statefulset"},
							},
							Operations: []admissionv1.OperationType{
								admissionv1.Create,
								admissionv1.Update,
							},
						},
						{
							Rule: admissionv1.Rule{
								APIGroups:   []string{"apps.openshift.io"},
								APIVersions: []string{"v1"},
								Resources:   []string{"deploymentconfigs"},
							},
							Operations: []admissionv1.OperationType{
								admissionv1.Create,
								admissionv1.Update,
							},
						},
					},
					ClientConfig: admissionv1.WebhookClientConfig{
						Service: &admissionv1.ServiceReference{
							Name:      wc.ComponentName(),
							Namespace: wc.Namespace,
							Port:      pointer.Int32Ptr(42650),
							Path:      pointer.String("/mutate"),
						},
						CABundle: wc.caCertPem,
					},
					AdmissionReviewVersions: []string{"v1"},
					SideEffects:             &sideEffect,
				},
			},
		},
	)
}

func NewWebhookConfigBuilder(
	agent *instanav1.InstanaAgent,
	isOpenShift bool,
	caCertPem []byte,
) builder.ObjectBuilder {
	return &webhookConfigBuilder{
		InstanaAgent: agent,
		helpers:      helpers.NewHelpers(agent),
		isOpenShift:  isOpenShift,
		caCertPem:    caCertPem,
	}
}
