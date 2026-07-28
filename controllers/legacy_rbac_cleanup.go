/*
(c) Copyright IBM Corp. 2026

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

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/client"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
)

const legacyEtcdReaderSuffix = "-etcd-reader"

// cleanupLegacyEtcdReaderRBAC removes the k8sensor etcd-reader Role and RoleBinding that
// earlier operator versions created in kube-system. They granted the k8sensor
// ServiceAccount permissions the k8sensor ClusterRole already grants cluster-wide, and
// they carried an ownerReference to an InstanaAgent CR in another namespace. Kubernetes
// rejects cross-namespace owner references, so the garbage collector deleted them and the
// operator recreated them on every reconcile.
//
// Leftovers that still carry that invalid ownerReference are removed by the garbage
// collector on its own once the operator stops recreating them. Deleting them explicitly
// also covers copies that lost the ownerReference, and makes the cleanup independent of
// the dependents ConfigMap, which only records them for the generation that created them.
//
// It reports whether both objects are confirmed gone, so that the caller can stop
// retrying. This can be dropped once installs have upgraded past the release that removed
// the builders.
func cleanupLegacyEtcdReaderRBAC(
	ctx context.Context,
	c client.InstanaAgentClient,
	agent *instanav1.InstanaAgent,
	log logr.Logger,
) bool {
	name := helpers.NewHelpers(agent).K8sSensorResourcesName() + legacyEtcdReaderSuffix
	objectMeta := metav1.ObjectMeta{Name: name, Namespace: kubeSystemNamespace}
	allGone := true

	for _, obj := range []k8sclient.Object{
		&rbacv1.Role{
			TypeMeta:   metav1.TypeMeta{APIVersion: rbacv1.SchemeGroupVersion.String(), Kind: "Role"},
			ObjectMeta: objectMeta,
		},
		&rbacv1.RoleBinding{
			TypeMeta:   metav1.TypeMeta{APIVersion: rbacv1.SchemeGroupVersion.String(), Kind: "RoleBinding"},
			ObjectMeta: objectMeta,
		},
	} {
		kind := obj.GetObjectKind().GroupVersionKind().Kind

		switch err := c.Delete(ctx, obj); {
		case err == nil:
			log.Info(
				"removed legacy k8sensor etcd-reader RBAC left behind by a previous operator version",
				"name",
				name,
				"namespace",
				kubeSystemNamespace,
				"kind",
				kind,
			)
		case !apierrors.IsNotFound(err):
			// Deliberately not logged at Error: this is best-effort cleanup of a previous
			// version's objects, nothing downstream depends on it, and a persistent
			// condition such as a Forbidden in a locked down cluster would otherwise
			// raise an alarm on every reconcile that nobody can act on.
			log.V(1).Info(
				"could not remove legacy k8sensor etcd-reader RBAC",
				"name", name,
				"namespace", kubeSystemNamespace,
				"kind", kind,
				"error", err,
			)

			allGone = false
		}
	}

	return allGone
}

// cleanupLegacyEtcdReaderRBACOnce runs cleanupLegacyEtcdReaderRBAC until it reports both
// objects gone, and never again for that agent while this operator process lives.
//
// Retrying forever would keep issuing deletes against a namespace the operator does not
// own, which is both permanent audit log noise and a way to fight anything that
// legitimately recreates an object under that name: a GitOps tool that had the churning
// Role pinned in its repository would be stuck recreating what the operator keeps
// deleting, which is the very symptom this change exists to end.
func (r *InstanaAgentReconciler) cleanupLegacyEtcdReaderRBACOnce(
	ctx context.Context,
	agent *instanav1.InstanaAgent,
	log logr.Logger,
) {
	key := k8sclient.ObjectKeyFromObject(agent)

	if _, done := r.legacyEtcdReaderCleaned.Load(key); done {
		return
	}

	if cleanupLegacyEtcdReaderRBAC(ctx, r.client, agent, log) {
		r.legacyEtcdReaderCleaned.Store(key, struct{}{})
	}
}
