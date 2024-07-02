/*
(c) Copyright IBM Corp. 2024
(c) Copyright Instana Inc.

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

package builder

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/instana/instana-agent-operator/pkg/k8s/object/transformations"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

type OptionalObject = optional.Optional[client.Object]

type ObjectBuilder interface {
	Build() OptionalObject
	ComponentName() string
	IsNamespaced() bool
}

func NewBuilderTransformer(transformations transformations.Transformations) BuilderTransformer {
	return &builderTransformer{
		transformations,
	}
}

type BuilderTransformer interface {
	Apply(bldr ObjectBuilder) OptionalObject
}

type builderTransformer struct {
	transformations transformations.Transformations
}

func (b *builderTransformer) Apply(builder ObjectBuilder) optional.Optional[client.Object] {
	switch opt := builder.Build(); opt.IsPresent() {
	case true:
		obj := opt.Get()
		b.transformations.AddCommonLabels(obj, builder.ComponentName())
		if builder.IsNamespaced() {
			b.transformations.AddOwnerReference(obj)
		}
		return optional.Of(obj)
	default:
		return opt
	}
}
