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
package transformations

import (
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/env"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/or_die"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

// labels
const (
	NameLabel       = "app.kubernetes.io/name"
	InstanceLabel   = "app.kubernetes.io/instance"
	VersionLabel    = "app.kubernetes.io/version"
	ComponentLabel  = "app.kubernetes.io/component"
	PartOfLabel     = "app.kubernetes.io/part-of"
	ManagedByLabel  = "app.kubernetes.io/managed-by"
	GenerationLabel = "agent.instana.io/generation"
)

const (
	name       = "instana-agent"
	partOf     = "instana"
	managedBy  = "instana-agent-operator"
	nameRemote = "remote-instana-agent"
)

var (
	version = env.GetOperatorVersion()
)

func GetVersion() string {
	return version
}

type Transformations interface {
	AddCommonLabels(obj client.Object, component string)
	AddOwnerReference(obj client.Object)
	PreviousGenerationsSelector() labels.Selector
}

type transformations struct {
	metav1.OwnerReference
	generation string
}

func (t *transformations) AddCommonLabels(obj client.Object, component string) {
	objLabels := optional.Of(obj.GetLabels()).GetOrDefault(make(map[string]string, 7))

	objLabels[NameLabel] = name
	objLabels[InstanceLabel] = t.Name
	objLabels[VersionLabel] = version
	objLabels[ComponentLabel] = component
	objLabels[PartOfLabel] = partOf
	objLabels[ManagedByLabel] = managedBy
	objLabels[GenerationLabel] = t.generation

	obj.SetLabels(objLabels)
}

func (t *transformations) PreviousGenerationsSelector() labels.Selector {
	return or_die.New[labels.Selector]().ResultOrDie(
		func() (labels.Selector, error) {
			return metav1.LabelSelectorAsSelector(
				&metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      NameLabel,
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{name},
						},
						{
							Key:      InstanceLabel,
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{t.Name},
						},
						{
							Key:      GenerationLabel,
							Operator: metav1.LabelSelectorOpNotIn,
							Values:   []string{t.generation},
						},
					},
				},
			)
		},
	)
}

func (t *transformations) AddOwnerReference(obj client.Object) {
	for _, preExisting := range obj.GetOwnerReferences() {
		if preExisting.UID == t.OwnerReference.UID {
			return
		}
	}

	obj.SetOwnerReferences(
		append(
			obj.GetOwnerReferences(),
			t.OwnerReference,
		),
	)
}

func NewTransformations(agent *instanav1.InstanaAgent) Transformations {
	return &transformations{
		OwnerReference: metav1.OwnerReference{
			APIVersion:         agent.APIVersion,
			Kind:               agent.Kind,
			Name:               agent.Name,
			UID:                agent.UID,
			Controller:         pointer.To(true),
			BlockOwnerDeletion: pointer.To(true),
		},
		generation: strconv.Itoa(int(agent.Generation)),
	}
}

func NewTransformationsRemote(agent *instanav1.RemoteAgent) Transformations {
	return &transformations{
		OwnerReference: metav1.OwnerReference{
			APIVersion:         agent.APIVersion,
			Kind:               agent.Kind,
			Name:               agent.Name,
			UID:                agent.UID,
			Controller:         pointer.To(true),
			BlockOwnerDeletion: pointer.To(true),
		},
		generation: strconv.Itoa(int(agent.Generation)),
	}
}
