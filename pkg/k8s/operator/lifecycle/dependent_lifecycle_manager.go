/*
(c) Copyright IBM Corp. 2024, 2025
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
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/collections/list"
	instanaClient "github.com/instana/instana-agent-operator/pkg/k8s/client"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/transformations"
	"github.com/instana/instana-agent-operator/pkg/multierror"
)

// DependentLifecycleManager is responsible of adding and removing dependents
// from the ConfigMap.Data field
type DependentLifecycleManager interface {
	UpdateDependentLifecycleInfo(currentGenerationDependents []client.Object) error
	CleanupDependents(currentDependents ...client.Object) error
}

type dependentLifecycleManager struct {
	ctx                context.Context
	agent              *instanav1.InstanaAgent
	instanaAgentClient instanaClient.InstanaAgentClient
	transformations    transformations.Transformations
}

func NewDependentLifecycleManager(
	ctx context.Context,
	agent *instanav1.InstanaAgent,
	instanaClient instanaClient.InstanaAgentClient,
) DependentLifecycleManager {
	return &dependentLifecycleManager{
		ctx:                ctx,
		agent:              agent,
		instanaAgentClient: instanaClient,
		transformations:    transformations.NewTransformations(agent),
	}
}

// UpdateDependentLifecycleInfo is responsible for adding all dependents listed
// in currentGenerationDependents to the ConfigMap.Data field and applying it
// through the InstanaAgentClient
func (d *dependentLifecycleManager) UpdateDependentLifecycleInfo(
	currentGenerationDependents []client.Object,
) error {
	log := logf.FromContext(d.ctx)

	retryError := d.retry(5, 100*time.Millisecond,
		func() (bool, error) {
			// Get the initial ConfigMap
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

			// Try to apply the changes
			_, err := d.instanaAgentClient.Apply(d.ctx, &lifecycleCm).Get()

			// If successful or error is not a conflict, break the loop
			if err == nil || !k8serrors.IsConflict(err) {
				return true, err
			}

			// If we got a conflict error, log and retry with exponential backoff
			log.Info("Conflict detected when updating ConfigMap, retrying...",
				"configmap", d.getConfigMapName(),
				"namespace", d.agent.GetNamespace(),
			)

			return false, err
		})

	if retryError != nil && k8serrors.IsConflict(retryError) {
		log.Error(retryError, "Failed to update ConfigMap after maximum retries due to conflicts",
			"configmap", d.getConfigMapName(),
			"namespace", d.agent.GetNamespace())
	}

	return retryError
}

// CleanupDependents is responsible of deleting all dependents that don't appear
// in the list whitelistedDependents from the ConfigMap.Data field and applying
// changes through the InstanaAgentClient
func (d *dependentLifecycleManager) CleanupDependents(
	currentDependents ...client.Object,
) error {
	log := logf.FromContext(d.ctx)

	retryError := d.retry(5, 100*time.Millisecond,
		func() (bool, error) {
			// Get the latest ConfigMap on each attempt
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
				_, deleteErr := d.deleteAll(deprecatedDependents)
				if deleteErr != nil {
					errBuilder.AddSingle(deleteErr)
				}

				if deleteErr == nil && key != d.getCurrentGenKey() {
					delete(lifecycleConfigMap.Data, key)
				}
			}

			result := d.instanaAgentClient.Apply(d.ctx, &lifecycleConfigMap)
			result.OnFailure(errBuilder.AddSingle)

			err := errBuilder.Build()

			// If successful or error is not a conflict, break the loop
			if err == nil || !isConflictError(err) {
				return true, err
			}

			// If we got a conflict error, log and retry with exponential backoff
			log.Info("Conflict detected when cleaning up dependents, retrying...",
				"configmap", d.getConfigMapName(),
				"namespace", d.agent.GetNamespace())

			return false, err
		})

	if retryError != nil && isConflictError(retryError) {
		log.Error(retryError, "Failed to clean up dependents after maximum retries due to conflicts",
			"configmap", d.getConfigMapName(),
			"namespace", d.agent.GetNamespace())
	}

	return retryError
}

// Retry is a simple retry function with a backtrack
func (d *dependentLifecycleManager) retry(attempts int, sleep time.Duration, fn func() (bool, error),
) error {
	var err error
	for i := range attempts {
		done, err := fn()
		if done {
			return err
		}

		// Skip sleeping after the last retry
		if i < attempts-1 {
			time.Sleep(sleep * time.Duration(1<<i))
		}
	}

	return err
}

// isConflictError checks if the error is a conflict error
func isConflictError(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's a direct conflict error
	if k8serrors.IsConflict(err) {
		return true
	}

	// Check if it contains a conflict error message
	return err.Error() != "" &&
		(err.Error() == "Operation cannot be fulfilled" ||
			err.Error() == "the object has been modified")
}

func (d *dependentLifecycleManager) getConfigMapName() string {
	return d.agent.GetName() + "-dependents"
}

func (d *dependentLifecycleManager) getCurrentGenKey() string {
	return fmt.Sprintf("%s_%d", transformations.GetVersion(), d.agent.GetGeneration())
}

func (d *dependentLifecycleManager) getOrInitializeLifecycleCm() (corev1.ConfigMap, error) {
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

func (d *dependentLifecycleManager) getLifecycleConfigMap() (corev1.ConfigMap, error) {
	lifecycleCm := corev1.ConfigMap{}
	err := d.instanaAgentClient.Get(
		d.ctx,
		types.NamespacedName{Name: d.getConfigMapName(), Namespace: d.agent.GetNamespace()},
		&lifecycleCm,
	)
	return lifecycleCm, err
}

func (d *dependentLifecycleManager) unmarshalToUnstructured(jsonString string) ([]unstructured.Unstructured, error) {
	var unstructuredData []unstructured.Unstructured
	err := json.Unmarshal([]byte(jsonString), &unstructuredData)
	return unstructuredData, err
}

func (d *dependentLifecycleManager) deleteAll(toBeDeleted []unstructured.Unstructured) ([]client.Object, error) {
	return d.instanaAgentClient.DeleteAllInTimeLimit(
		d.ctx,
		list.
			NewListMapTo[unstructured.Unstructured, client.Object]().
			MapTo(toBeDeleted, asObject),
		30*time.Second,
		5*time.Second).Get()
}
