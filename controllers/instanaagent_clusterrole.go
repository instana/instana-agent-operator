/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package controllers

import (
	"context"
	"fmt"

	rbacV1 "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newClusterRoleForCRD() *rbacV1.ClusterRole {
	return &rbacV1.ClusterRole{
		ObjectMeta: metaV1.ObjectMeta{
			Name:   AppName,
			Labels: buildLabels(),
		},
		Rules: []rbacV1.PolicyRule{
			{
				NonResourceURLs: []string{"/version"},
				Verbs:           []string{"get"},
			},
			{
				APIGroups: []string{"batch"},
				Resources: []string{"jobs", "cronjobs"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"extensions"},
				Resources: []string{"deployments", "replicasets", "ingresses"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"apps"},
				Resources: []string{"deployments", "replicasets", "daemonsets", "statefulsets"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"namespaces", "events", "services", "endpoints", "nodes", "pods", "replicationcontrollers",
					"componentstatuses", "resourcequotas", "persistentvolumes", "persistentvolumeclaims"},
				Verbs: []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"endpoints"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"networking.k8s.io"},
				Resources: []string{"ingresses"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}
}

func (r *InstanaAgentReconciler) reconcileClusterRole(ctx context.Context) error {
	clusterRole := &rbacV1.ClusterRole{}
	err := r.Get(ctx, client.ObjectKey{Name: AppName}, clusterRole)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			r.Log.Info("No InstanaAgent clusterRole deployed before, creating new one")
			clusterRole = newClusterRoleForCRD()
			if err = r.Create(ctx, clusterRole); err == nil {
				r.Log.Info(fmt.Sprintf("%s clusterRole created successfully", AppName))
				return nil
			} else {
				r.Log.Error(err, "Failed to create Instana agent clusterRole")
			}
		}
		return err
	}
	return nil
}
