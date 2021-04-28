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

func newClusterRoleBindingForCRD() *rbacV1.ClusterRoleBinding {
	return &rbacV1.ClusterRoleBinding{
		ObjectMeta: metaV1.ObjectMeta{
			Name:   AppName,
			Labels: buildLabels(),
		},
		Subjects: []rbacV1.Subject{{
			Kind:      "ServiceAccount",
			Name:      AppName,
			Namespace: AgentNameSpace,
		}},
		RoleRef: rbacV1.RoleRef{
			Kind:     "ClusterRole",
			Name:     AppName,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
}

func (r *InstanaAgentReconciler) reconcileClusterRoleBinding(ctx context.Context) error {
	clusterRoleBinding := &rbacV1.ClusterRoleBinding{}
	err := r.Get(ctx, client.ObjectKey{Name: AppName}, clusterRoleBinding)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			r.Log.Info("No InstanaAgent clusterRoleBinding deployed before, creating new one")
			clusterRoleBinding = newClusterRoleBindingForCRD()
			if err = r.Create(ctx, clusterRoleBinding); err == nil {
				r.Log.Info(fmt.Sprintf("%s clusterRoleBinding created successfully", AppName))
				return nil
			} else {
				r.Log.Error(err, "Failed to create Instana agent clusterRoleBinding")
			}
		}
		return err
	}
	return nil
}
