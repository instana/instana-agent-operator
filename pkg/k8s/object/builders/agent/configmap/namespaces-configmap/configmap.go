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

package namespaces_configmap

import (
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/namespaces"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/status"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/or_die"
)

type namespacesConfigMapBuilder struct {
	*instanav1.InstanaAgent
	statusManager     status.AgentStatusManager
	namespacesDetails namespaces.NamespacesDetails
}

func (c *namespacesConfigMapBuilder) ComponentName() string {
	return constants.ComponentInstanaAgent
}

func (c *namespacesConfigMapBuilder) IsNamespaced() bool {
	return true
}

func yamlOrDie(obj any) string {
	return string(
		or_die.New[[]byte]().
			ResultOrDie(
				func() ([]byte, error) {
					return yaml.Marshal(obj)
				},
			),
	)
}

func (c *namespacesConfigMapBuilder) getData() map[string]string {
	res := make(map[string]string)

	res["namespaces.yaml"] = yamlOrDie(&c.namespacesDetails)

	return res

}

func (c *namespacesConfigMapBuilder) Build() (res optional.Optional[client.Object]) {
	defer func() {
		res.IfPresent(
			func(cm client.Object) {
				c.statusManager.SetAgentNamespacesConfigMap(client.ObjectKeyFromObject(cm))
			},
		)
	}()

	return optional.Of[client.Object](
		&corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      c.Name + "-namespaces",
				Namespace: c.Namespace,
			},
			Data: c.getData(),
		},
	)
}

func NewConfigMapBuilder(agent *instanav1.InstanaAgent, statusManager status.AgentStatusManager, namespacesDetails namespaces.NamespacesDetails) builder.ObjectBuilder {
	return &namespacesConfigMapBuilder{
		InstanaAgent:      agent,
		statusManager:     statusManager,
		namespacesDetails: namespacesDetails,
	}
}
