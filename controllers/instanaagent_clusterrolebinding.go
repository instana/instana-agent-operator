/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package controllers

import (
	rbacV1 "k8s.io/api/rbac/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
