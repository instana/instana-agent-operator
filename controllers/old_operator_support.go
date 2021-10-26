/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package controllers

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	appV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ManagedByLabelKey       = "app.kubernetes.io/managed-by"
	HelmReleaseNameKey      = "meta.helm.sh/release-name"
	HelmReleaseNamespaceKey = "meta.helm.sh/release-namespace"
)

func (r *InstanaAgentReconciler) getAndDeleteOldOperator(ctx context.Context) (bool, error) {
	oldOperatorDeployment := &appV1.Deployment{}
	if err := r.client.Get(ctx, client.ObjectKey{
		Namespace: "instana-agent",
		Name:      "instana-agent-operator",
	}, oldOperatorDeployment); err != nil {
		if k8sErrors.IsNotFound(err) {
			r.log.V(1).Info("No old Operator Deployment found, not necessary to delete")
			return false, nil
		} else {
			r.log.Error(err, "Failure looking for old Operator Deployment")
			return false, err
		}
	}

	r.log.V(1).Info(fmt.Sprintf("Found old Operator Deployment and will try to delete: %v", oldOperatorDeployment))
	if err := r.client.Delete(ctx, oldOperatorDeployment); err != nil {
		r.log.Error(err, "Failure deleting old Operator Deployment")
		return false, err
	}

	return true, nil
}

type ObjectListItemsConversion struct {
	list client.ObjectList
}

func (c *ObjectListItemsConversion) getClientObjectItems() []client.Object {
	var convertedItems []client.Object

	switch objType := c.list.(type) {
	case *rbacv1.ClusterRoleBindingList:
		convertedItems = make([]client.Object, len(objType.Items))
		for i := range objType.Items {
			convertedItems[i] = &objType.Items[i]
		}
	case *rbacv1.ClusterRoleList:
		convertedItems = make([]client.Object, len(objType.Items))
		for i := range objType.Items {
			convertedItems[i] = &objType.Items[i]
		}
	case *coreV1.ServiceAccountList:
		convertedItems = make([]client.Object, len(objType.Items))
		for i := range objType.Items {
			convertedItems[i] = &objType.Items[i]
		}
	case *coreV1.ConfigMapList:
		convertedItems = make([]client.Object, len(objType.Items))
		for i := range objType.Items {
			convertedItems[i] = &objType.Items[i]
		}
	case *coreV1.SecretList:
		convertedItems = make([]client.Object, len(objType.Items))
		for i := range objType.Items {
			convertedItems[i] = &objType.Items[i]
		}
	case *appV1.DaemonSetList:
		convertedItems = make([]client.Object, len(objType.Items))
		for i := range objType.Items {
			convertedItems[i] = &objType.Items[i]
		}
	}

	return convertedItems
}

// getAndDeleteOldOperatorResources removes all resources that might exist and belong to the 'old' (Java) operator.
// Initially tried adding labels and annotations so Helm could inherit these resources, but that mostly led to deployment issues.
// So instead just completely removing so Helm can do a 'clean' install. Downside is a short interruption of the Agent, but if
// the DaemonSet was not exactly the same, chances are very high it would get re-deployed anyhow by Helm.
func (r *InstanaAgentReconciler) getAndDeleteOldOperatorResources(ctx context.Context) (bool, error) {
	// List of object types that need possible deletion.
	// Make sure to add any type also to the ObjectListItemsConversion.getClientObjectItems method
	toDeleteResourceTypes := []ObjectListItemsConversion{
		{list: &rbacv1.ClusterRoleBindingList{}},
		{list: &rbacv1.ClusterRoleList{}},
		{list: &coreV1.ServiceAccountList{}},
		{list: &coreV1.ConfigMapList{}},
		{list: &coreV1.SecretList{}},
		{list: &appV1.DaemonSetList{}},
	}

	// Init and add to selector.
	selector := labels.NewSelector()
	managedByReq, _ := labels.NewRequirement(ManagedByLabelKey, selection.Equals, []string{"instana-agent-operator"})
	selector = selector.Add(*managedByReq)

	hasDeletedResources := false
	for _, toBeDeletedType := range toDeleteResourceTypes {
		if deleted, err := r.deleteResourcesOfType(ctx, toBeDeletedType, selector); err != nil {
			return false, err
		} else if deleted {
			hasDeletedResources = true
		}
	}

	return hasDeletedResources, nil
}

func (r *InstanaAgentReconciler) deleteResourcesOfType(ctx context.Context, toBeDeletedType ObjectListItemsConversion, selector labels.Selector) (bool, error) {

	if err := r.client.List(ctx, toBeDeletedType.list, client.MatchingLabelsSelector{Selector: selector}); err != nil {
		r.log.Error(err, fmt.Sprintf("Failure listing: %v", toBeDeletedType))
		return false, err
	}

	resourceInstances := toBeDeletedType.getClientObjectItems()

	for _, resource := range resourceInstances {
		if err := r.deleteResource(ctx, resource); err != nil {
			return false, err
		}
	}

	return len(resourceInstances) > 0, nil
}

func (r *InstanaAgentReconciler) deleteResource(ctx context.Context, item client.Object) error {
	r.log.V(1).Info(fmt.Sprintf("Found existing resource managed by old Operator, trying to delete: %v", item))
	// client.DeleteAllOf doesn't work for all types of resources, so just delete them one by one.
	if err := r.client.Delete(ctx, item); err != nil {
		r.log.Error(err, fmt.Sprintf("Failure deleting existing resource: %v", item))
		return err
	}
	return nil
}
