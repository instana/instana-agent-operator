/*
(c) Copyright IBM Corp. 2024
(c) Copyright Instana Inc.
*/

package configmap

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

type serviceCaConfigMapBuilder struct {
	*instanav1.InstanaAgent
	helpers.Helpers
}

func (c *serviceCaConfigMapBuilder) IsNamespaced() bool {
	return true
}

func (c *serviceCaConfigMapBuilder) ComponentName() string {
	return constants.ComponentK8Sensor
}

func (c *serviceCaConfigMapBuilder) Build() optional.Optional[client.Object] {
	return optional.Of[client.Object](
		&corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sensor-service-ca",
				Namespace: c.Namespace,
				Annotations: map[string]string{
					"service.beta.openshift.io/inject-cabundle": "true",
				},
			},
			Data: map[string]string{},
		})
}

func NewServiceCaConfigMapBuilder(agent *instanav1.InstanaAgent) builder.ObjectBuilder {
	return &serviceCaConfigMapBuilder{
		InstanaAgent: agent,
		Helpers:      helpers.NewHelpers(agent),
	}
}
