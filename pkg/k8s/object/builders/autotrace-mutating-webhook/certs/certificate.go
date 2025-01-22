/*
(c) Copyright IBM Corp. 2025
(c) Copyright Instana Inc. 2025
*/

package certs

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

type certBuilder struct {
	*instanav1.InstanaAgent
	helpers     helpers.Helpers
	isOpenShift bool
	certPem     []byte
	keyPem      []byte
}

func (c *certBuilder) IsNamespaced() bool {
	return true
}

func (c *certBuilder) ComponentName() string {
	return c.helpers.AutotraceWebhookResourcesName() + "-certs"
}

func (c *certBuilder) Build() (res optional.Optional[client.Object]) {

	if c.isOpenShift {
		return optional.Empty[client.Object]()
	}

	return optional.Of[client.Object](
		&corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      c.ComponentName(),
				Namespace: c.Namespace,
				//todo: add labels
			},
			Data: map[string][]byte{
				"tls.crt": c.certPem,
				"tls.key": c.keyPem,
				"ca.crt":  c.certPem,
			},
		},
	)
}

func NewCertBuilder(
	agent *instanav1.InstanaAgent,
	isOpenShift bool,
	certPem []byte,
	keyPem []byte,
) builder.ObjectBuilder {
	return &certBuilder{
		InstanaAgent: agent,
		helpers:      helpers.NewHelpers(agent),
		isOpenShift:  isOpenShift,
		certPem:      certPem,
		keyPem:       keyPem,
	}
}
