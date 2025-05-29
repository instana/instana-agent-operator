/*
(c) Copyright IBM Corp. 2025

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package helpers

import (
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

type remoteHelpers struct {
	*instanav1.RemoteAgent
}

type RemoteHelpers interface {
	ServiceAccountName() string
	TLSIsEnabled() bool
	TLSSecretName() string
	HeadlessServiceName() string
	K8sSensorResourcesName() string
	ContainersSecretName() string
	UseContainersSecret() bool
	ImagePullSecrets() []corev1.LocalObjectReference
	SortEnvVarsByName(envVars []corev1.EnvVar)
}

func (h *remoteHelpers) serviceAccountNameDefault() string {
	switch pointer.DerefOrEmpty(h.Spec.ServiceAccountSpec.Create.Create) {
	case true:
		return h.Name
	default:
		return "default"
	}
}

func (h *remoteHelpers) ServiceAccountName() string {
	return optional.Of(h.Spec.ServiceAccountSpec.Name.Name).GetOrDefault(h.serviceAccountNameDefault())
}

func (h *remoteHelpers) TLSIsEnabled() bool {
	return h.Spec.Agent.TlsSpec.SecretName != "" || (len(h.Spec.Agent.TlsSpec.Certificate) > 0 && len(h.Spec.Agent.TlsSpec.Key) > 0)
}

func (h *remoteHelpers) TLSSecretName() string {
	return optional.Of(h.Spec.Agent.TlsSpec.SecretName).GetOrDefault(h.Name + "-tls")
}

func (h *remoteHelpers) HeadlessServiceName() string {
	return h.Name + "-headless"
}

func (h *remoteHelpers) ContainersSecretName() string {
	return h.Name + "-containers-instana-io"
}

func (h *remoteHelpers) UseContainersSecret() bool {
	// Explicitly using e.PullSecrets != nil instead of len(e.PullSecrets) == 0 since the original chart specified that
	// auto-generated secret shouldn't be used if the user explicitly provided an empty list of pull secrets
	// (original logic was to only use the generated secret if the registry matches AND the pullSecrets field was
	// omitted by the user). I don't understand why anyone would want this, but the original chart had comments
	// specifically mentioning that this was the desired behavior, so keeping it until someone says otherwise.
	return h.Spec.Agent.PullSecrets == nil && strings.HasPrefix(
		h.Spec.Agent.ImageSpec.Name,
		ContainersInstanaIORegistry,
	)
}

func (h *remoteHelpers) ImagePullSecrets() []corev1.LocalObjectReference {
	if h.UseContainersSecret() {
		return []corev1.LocalObjectReference{
			{
				Name: h.ContainersSecretName(),
			},
		}
	} else {
		return h.Spec.Agent.ExtendedImageSpec.PullSecrets
	}
}

func (h *remoteHelpers) K8sSensorResourcesName() string {
	return h.Name + "-k8sensor"
}

func (h *remoteHelpers) SortEnvVarsByName(envVars []corev1.EnvVar) {
	sort.Slice(envVars, func(i, j int) bool {
		return envVars[i].Name < envVars[j].Name
	})
}

func NewRemoteHelpers(agent *instanav1.RemoteAgent) RemoteHelpers {
	return &remoteHelpers{
		RemoteAgent: agent,
	}
}
