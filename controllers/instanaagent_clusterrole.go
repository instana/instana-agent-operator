/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package controllers

import (
	rbacV1 "k8s.io/api/rbac/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
