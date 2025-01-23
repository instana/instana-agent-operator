/*
(c) Copyright IBM Corp. 2025
(c) Copyright Instana Inc. 2025
*/

package secrets

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	webhookconfig "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/autotrace-mutating-webhook/webhookconfig"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

type certBuilder struct {
	*instanav1.InstanaAgent
	helpers     helpers.Helpers
	isOpenShift bool
	chainPem    []byte
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

	leafPem, caPem, err := webhookconfig.ExtractLeafAndCa(c.chainPem)
	if err != nil {
		fmt.Println("error seperating the leaf and CA PEM")
		return optional.Empty[client.Object]()
	}

	return optional.Of[client.Object](
		&corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			Type: corev1.SecretTypeTLS,
			ObjectMeta: metav1.ObjectMeta{
				Name:      c.ComponentName(),
				Namespace: c.Namespace,
			},
			Data: map[string][]byte{
				"tls.crt": c.chainPem,
				"tls.key": c.keyPem,
				"ca.crt":  caPem,
			},
		},
	)
}

func NewCertBuilder(
	agent *instanav1.InstanaAgent,
	isOpenShift bool,
	chainPem []byte,
	keyPem []byte,
) builder.ObjectBuilder {
	return &certBuilder{
		InstanaAgent: agent,
		helpers:      helpers.NewHelpers(agent),
		isOpenShift:  isOpenShift,
		chainPem:     chainPem,
		keyPem:       keyPem,
	}
}
