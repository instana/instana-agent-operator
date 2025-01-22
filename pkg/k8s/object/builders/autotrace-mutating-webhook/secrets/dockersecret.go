/*
(c) Copyright IBM Corp. 2025
(c) Copyright Instana Inc. 2025
*/

package secrets

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

type downloadSecretBuilder struct {
	*instanav1.InstanaAgent
	helpers helpers.Helpers
}

func (s *downloadSecretBuilder) IsNamespaced() bool {
	return true
}

func (s *downloadSecretBuilder) ComponentName() string {
	return s.helpers.AutotraceWebhookResourcesName()
}

func (s *downloadSecretBuilder) getWebhookImagePullSecret() string {
	if s.InstanaAgent.Spec.AutotraceWebhook.PullSecret != "" {
		return s.InstanaAgent.Spec.AutotraceWebhook.PullSecret
	} else {
		return "containers-instana-io"
	}
}

func (s *downloadSecretBuilder) Build() (res optional.Optional[client.Object]) {

	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", "_", s.Spec.Agent.Key)))

	dockerConfig := map[string]interface{}{
		"auths": map[string]interface{}{
			s.getWebhookImagePullSecret(): map[string]string{
				"username": "_",
				"password": s.Spec.Agent.Key,
				"auth":     auth,
			},
		},
	}

	dockerConfigJson, err := json.Marshal(dockerConfig)
	if err != nil {
		fmt.Println("failed to marchal dockerconfig to JSON for webhook pull secret: %w", err)
		return nil
	}

	return optional.Of[client.Object](
		&corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			Type: corev1.SecretTypeDockerConfigJson,
			ObjectMeta: metav1.ObjectMeta{
				Name:      s.ComponentName(),
				Namespace: s.Namespace,
				//todo: add labels
			},
			Data: map[string][]byte{
				corev1.DockerConfigJsonKey: dockerConfigJson,
			},
		},
	)
}

func NewDownloadSecretBuilder(
	agent *instanav1.InstanaAgent,
) builder.ObjectBuilder {
	return &downloadSecretBuilder{
		InstanaAgent: agent,
		helpers:      helpers.NewHelpers(agent),
	}
}
