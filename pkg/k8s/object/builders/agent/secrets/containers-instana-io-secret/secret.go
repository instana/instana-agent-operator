/*
(c) Copyright IBM Corp. 2024
*/

package containers_instana_io_secret

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/json_or_die"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/optional"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("containers-instana-io-secret-builder")

type DockerConfigAuth struct {
	Auth []byte `json:"auth"`
}

type DockerConfigJson struct {
	Auths map[string]DockerConfigAuth `json:"auths"`
}

type secretBuilder struct {
	instanaAgent *instanav1.InstanaAgent
	helpers      helpers.Helpers
	marshaler    json_or_die.JsonOrDieMarshaler[*DockerConfigJson]
	keysSecret   *corev1.Secret
}

func NewSecretBuilder(agent *instanav1.InstanaAgent, keysSecret *corev1.Secret) builder.ObjectBuilder {
	return &secretBuilder{
		instanaAgent: agent,
		keysSecret:   keysSecret,
		helpers:      helpers.NewHelpers(agent),
		marshaler:    json_or_die.NewJsonOrDie[DockerConfigJson](),
	}
}

func (s *secretBuilder) IsNamespaced() bool {
	return true
}

func (s *secretBuilder) ComponentName() string {
	return constants.ComponentInstanaAgent
}

func (s *secretBuilder) Build() optional.Optional[client.Object] {
	switch s.helpers.UseContainersSecret() {
	case true:
		return optional.Of[client.Object](s.build())
	default:
		return optional.Empty[client.Object]()
	}
}

func (s *secretBuilder) build() *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.helpers.ContainersSecretName(),
			Namespace: s.instanaAgent.Namespace,
		},
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: s.buildDockerConfigJson(),
		},
		Type: corev1.SecretTypeDockerConfigJson,
	}
}

func (s *secretBuilder) buildDockerConfigJson() []byte {
	// prefer downloadKey over key property
	// prefer referenced secret over custom resource property
	var downloadKey string = ""
	if downloadKeyValueFromSecret, ok := s.keysSecret.Data["downloadKey"]; ok {
		downloadKey = string(downloadKeyValueFromSecret)
	} else if keyValueFromSecret, ok := s.keysSecret.Data["key"]; ok {
		downloadKey = string(keyValueFromSecret)
	} else if s.instanaAgent.Spec.Agent.DownloadKey != "" {
		downloadKey = s.instanaAgent.Spec.Agent.DownloadKey
	} else if s.instanaAgent.Spec.Agent.Key != "" {
		downloadKey = s.instanaAgent.Spec.Agent.Key
	} else {
		// we are lacking any download key information
		log.Error(fmt.Errorf("cannot extract download key from secret or custom resource"), "No download key available")
	}

	auth := fmt.Sprintf("_:%s", downloadKey)

	json := DockerConfigJson{
		Auths: map[string]DockerConfigAuth{
			helpers.ContainersInstanaIORegistry: {
				Auth: []byte(auth),
			},
		},
	}
	return s.marshaler.MarshalOrDie(&json)
}
