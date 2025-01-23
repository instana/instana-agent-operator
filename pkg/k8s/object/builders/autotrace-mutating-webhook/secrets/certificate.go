/*
(c) Copyright IBM Corp. 2025
(c) Copyright Instana Inc. 2025
*/

package secrets

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/pem"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/cert"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

type certBuilder struct {
	*instanav1.InstanaAgent
	helpers       helpers.Helpers
	isOpenShift   bool
	caCertPem     []byte
	serverCertPem []byte
	serverKeyPem  []byte
}

func (c *certBuilder) IsNamespaced() bool {
	return true
}

func (c *certBuilder) ComponentName() string {
	return c.helpers.AutotraceWebhookResourcesName() + "-certs"
}

func GenerateCerts() (caCertPem, serverCertPem, serverKeyPem []byte, err error) {
	// generate CA
	caKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	caConfig := cert.Config{
		CommonName: "instana-autotrace-webhook-ca",
	}
	caCert, _ := cert.NewSelfSignedCACert(caConfig, caKey)
	caCertPem = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caCert.Raw,
	})

	//generate server cert
	serverCertPem, serverKeyPem, err = cert.GenerateSelfSignedCertKey(
		"instana-autotrace-webhook",
		nil,
		[]string{
			"instana-autotrace-webhook.instana-agent",
			"instana-autotrace-webhook.instana-agent.svc",
			"instana-autotrace-webhook.instana-agent.svc.cluster.local",
		},
	)
	return caCertPem, serverCertPem, serverKeyPem, nil
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
			Type: corev1.SecretTypeTLS,
			ObjectMeta: metav1.ObjectMeta{
				Name:      c.ComponentName(),
				Namespace: c.Namespace,
			},
			Data: map[string][]byte{
				"tls.crt": c.serverCertPem,
				"tls.key": c.serverKeyPem,
				"ca.crt":  c.caCertPem,
			},
		},
	)
}

func NewCertBuilder(
	agent *instanav1.InstanaAgent,
	isOpenShift bool,
	caCertPem []byte,
	serverCertPem []byte,
	serverKeyPem []byte,
) builder.ObjectBuilder {
	return &certBuilder{
		InstanaAgent:  agent,
		helpers:       helpers.NewHelpers(agent),
		isOpenShift:   isOpenShift,
		serverCertPem: serverCertPem,
		serverKeyPem:  serverKeyPem,
		caCertPem:     caCertPem,
	}
}
