/*
(c) Copyright IBM Corp. 2024
(c) Copyright Instana Inc. 2024
*/

package configmap

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	backends "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/backends"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

type configMapBuilder struct {
	*instanav1.InstanaAgent
	helpers.Helpers
	backends []backends.K8SensorBackend
}

func (c *configMapBuilder) IsNamespaced() bool {
	return true
}

func (c *configMapBuilder) ComponentName() string {
	return constants.ComponentK8Sensor
}

func (c *configMapBuilder) getConfigMapData() map[string]string {
	data := make(map[string]string, len(c.backends))
	for _, backend := range c.backends {
		data[constants.BackendKey+backend.ResourceSuffix] = fmt.Sprintf("%s:%s", backend.EndpointHost, backend.EndpointPort)
	}
	return data
}

func (c *configMapBuilder) Build() optional.Optional[client.Object] {
	return optional.Of[client.Object](
		&corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      c.K8sSensorResourcesName(),
				Namespace: c.Namespace,
			},
			Data: c.getConfigMapData(),
		})
}

func NewConfigMapBuilder(agent *instanav1.InstanaAgent, backends []backends.K8SensorBackend) builder.ObjectBuilder {
	return &configMapBuilder{
		InstanaAgent: agent,
		Helpers:      helpers.NewHelpers(agent),
		backends:     backends,
	}
}
