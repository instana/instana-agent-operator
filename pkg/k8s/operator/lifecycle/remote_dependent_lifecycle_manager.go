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
package lifecycle

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/collections/list"
	instanaClient "github.com/instana/instana-agent-operator/pkg/k8s/client"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/transformations"
	"github.com/instana/instana-agent-operator/pkg/multierror"
)

// DependentLifecycleManager is responsible of adding and removing dependents
// from the ConfigMap.Data field
type RemoteDependentLifecycleManager interface {
	UpdateDependentLifecycleInfo(currentGenerationDependents []client.Object) error
	CleanupDependents(currentDependents ...client.Object) error
}

type remoteDependentLifecycleManager struct {
	ctx                context.Context
	agent              *instanav1.RemoteAgent
	instanaAgentClient instanaClient.InstanaAgentClient
	transformations    transformations.Transformations
}

func NewRemoteDependentLifecycleManager(
	ctx context.Context,
	agent *instanav1.RemoteAgent,
	instanaClient instanaClient.InstanaAgentClient,
) RemoteDependentLifecycleManager {
	return &remoteDependentLifecycleManager{
		ctx:                ctx,
		agent:              agent,
		instanaAgentClient: instanaClient,
		transformations:    transformations.NewTransformationsRemote(agent),
	}
}

// UpdateDependentLifecycleInfo is responsible for adding all dependents listed
// in currentGenerationDependents to the ConfigMap.Data field and applying it
// through the InstanaAgentClient
func (d *remoteDependentLifecycleManager) UpdateDependentLifecycleInfo(
	currentGenerationDependents []client.Object,
) error {
	lifecycleCm, _ := d.getOrInitializeLifecycleCm()
	currentGenKey := d.getCurrentGenKey()

	// Ensures that a lifecycle comparison will be performed even if neither the generation nor the operator version
	// have been updated, should only be necessary for the sake of testing during development
	if existingVersion, isPresent := lifecycleCm.Data[currentGenKey]; isPresent {
		lifecycleCm.Data[currentGenKey+"-dirty"] = existingVersion
	}

	currentDependentsJson, _ := json.Marshal(asUnstructureds(currentGenerationDependents...))
	lifecycleCm.Data[currentGenKey] = string(currentDependentsJson)

	d.transformations.AddCommonLabels(&lifecycleCm, constants.ComponentInstanaAgent)
	d.transformations.AddOwnerReference(&lifecycleCm)

	_, err := d.instanaAgentClient.Apply(d.ctx, &lifecycleCm).Get()

	return err
}

// CleanupDependents is responsible of deleting all dependents that don't appear
// in the list whitelistedDependents from the ConfigMap.Data field and applying
// changes through the InstanaAgentClient
func (d *remoteDependentLifecycleManager) CleanupDependents(
	currentDependents ...client.Object,
) error {
	lifecycleConfigMap, _ := d.getLifecycleConfigMap()
	errBuilder := multierror.NewMultiErrorBuilder()
	currentGeneration := asUnstructureds(currentDependents...)

	for key, jsonString := range lifecycleConfigMap.Data {
		olderGeneration, _ := d.unmarshalToUnstructured(jsonString)
		deprecatedDependents := list.
			NewDeepDiff[unstructured.Unstructured]().
			Diff(
				olderGeneration,
				currentGeneration,
			)
		_, err := d.deleteAll(deprecatedDependents)
		if err != nil {
			errBuilder.AddSingle(err)
		}

		if err == nil && key != d.getCurrentGenKey() {
			delete(lifecycleConfigMap.Data, key)
		}
	}

	d.instanaAgentClient.Apply(d.ctx, &lifecycleConfigMap).OnFailure(errBuilder.AddSingle)

	return errBuilder.Build()
}

func (d *remoteDependentLifecycleManager) getConfigMapName() string {
	return d.agent.GetName() + "-dependents"
}

func (d *remoteDependentLifecycleManager) getCurrentGenKey() string {
	return fmt.Sprintf("%s_%d", transformations.GetVersion(), d.agent.GetGeneration())
}

func (d *remoteDependentLifecycleManager) getOrInitializeLifecycleCm() (corev1.ConfigMap, error) {
	lifecycleConfigMap, err := d.getLifecycleConfigMap()

	// Initialize ConfigMap if we weren't able to fetch one
	if err != nil {
		if k8serrors.IsNotFound(err) {
			lifecycleConfigMap = corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      d.getConfigMapName(),
					Namespace: d.agent.GetNamespace(),
				},
			}
		} else {
			lifecycleConfigMap = corev1.ConfigMap{}
		}
	}

	// If the Data field is empty, set a value so that we wont die when attempting to read
	if lifecycleConfigMap.Data == nil {
		lifecycleConfigMap.Data = make(map[string]string, 1)
	}

	return lifecycleConfigMap, err
}

func (d *remoteDependentLifecycleManager) getLifecycleConfigMap() (corev1.ConfigMap, error) {
	lifecycleCm := corev1.ConfigMap{}
	err := d.instanaAgentClient.Get(
		d.ctx,
		types.NamespacedName{Name: d.getConfigMapName(), Namespace: d.agent.GetNamespace()},
		&lifecycleCm,
	)
	return lifecycleCm, err
}

func (d *remoteDependentLifecycleManager) unmarshalToUnstructured(jsonString string) ([]unstructured.Unstructured, error) {
	var unstructuredData []unstructured.Unstructured
	err := json.Unmarshal([]byte(jsonString), &unstructuredData)
	return unstructuredData, err
}

func (d *remoteDependentLifecycleManager) deleteAll(toBeDeleted []unstructured.Unstructured) ([]client.Object, error) {
	return d.instanaAgentClient.DeleteAllInTimeLimit(
		d.ctx,
		list.
			NewListMapTo[unstructured.Unstructured, client.Object]().
			MapTo(toBeDeleted, asObject),
		30*time.Second,
		5*time.Second).Get()
}
