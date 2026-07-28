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
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	instanaclient "github.com/instana/instana-agent-operator/pkg/k8s/client"
)

const legacyEtcdReaderName = "instana-agent-k8sensor-etcd-reader"

// deleteCountingClient counts Delete calls, and optionally fails them, so that the
// retry behaviour of cleanupLegacyEtcdReaderRBACOnce can be observed.
type deleteCountingClient struct {
	instanaclient.InstanaAgentClient
	deletes int
	err     error
}

func (d *deleteCountingClient) Delete(
	ctx context.Context,
	obj k8sclient.Object,
	opts ...k8sclient.DeleteOption,
) error {
	d.deletes++

	if d.err != nil {
		return d.err
	}

	return d.InstanaAgentClient.Delete(ctx, obj, opts...)
}

func legacyEtcdReaderAgent() *instanav1.InstanaAgent {
	return &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "instana-agent",
			Namespace: "base-mon",
		},
	}
}

func legacyEtcdReaderRole(namespace string) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      legacyEtcdReaderName,
			Namespace: namespace,
		},
	}
}

func legacyEtcdReaderRoleBinding(namespace string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      legacyEtcdReaderName,
			Namespace: namespace,
		},
	}
}

// The etcd-reader Role and RoleBinding a previous operator version created in kube-system
// must not survive an upgrade, even when the dependents ConfigMap no longer records them.
func TestCleanupLegacyEtcdReaderRBAC(t *testing.T) {
	for _, tc := range []struct {
		name            string
		existingObjects []runtime.Object
	}{
		{
			name: "both_objects_present",
			existingObjects: []runtime.Object{
				legacyEtcdReaderRole(kubeSystemNamespace),
				legacyEtcdReaderRoleBinding(kubeSystemNamespace),
			},
		},
		{
			name: "only_role_present",
			existingObjects: []runtime.Object{
				legacyEtcdReaderRole(kubeSystemNamespace),
			},
		},
		{
			name:            "nothing_left_to_clean_up",
			existingObjects: []runtime.Object{},
		},
	} {
		t.Run(
			tc.name, func(t *testing.T) {
				assertions := require.New(t)

				scheme := runtime.NewScheme()
				assertions.NoError(rbacv1.AddToScheme(scheme))

				baseClient := fake.NewClientBuilder().
					WithScheme(scheme).
					WithRuntimeObjects(tc.existingObjects...).
					Build()

				agent := legacyEtcdReaderAgent()

				cleanupLegacyEtcdReaderRBAC(
					context.Background(),
					instanaclient.NewInstanaAgentClient(baseClient),
					agent,
					zap.New(),
				)

				key := types.NamespacedName{
					Name:      legacyEtcdReaderName,
					Namespace: kubeSystemNamespace,
				}

				assertions.True(
					apierrors.IsNotFound(
						baseClient.Get(context.Background(), key, &rbacv1.Role{}),
					),
					"legacy etcd-reader Role should no longer exist in kube-system",
				)
				assertions.True(
					apierrors.IsNotFound(
						baseClient.Get(context.Background(), key, &rbacv1.RoleBinding{}),
					),
					"legacy etcd-reader RoleBinding should no longer exist in kube-system",
				)
			},
		)
	}
}

// Retrying forever would fight anything that legitimately recreates an object under that
// name, so once the objects are gone the operator must stop issuing deletes.
func TestCleanupLegacyEtcdReaderRBACOnceStopsAfterTheObjectsAreGone(t *testing.T) {
	assertions := require.New(t)

	scheme := runtime.NewScheme()
	assertions.NoError(rbacv1.AddToScheme(scheme))

	agent := legacyEtcdReaderAgent()

	countingClient := &deleteCountingClient{
		InstanaAgentClient: instanaclient.NewInstanaAgentClient(
			fake.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(
					legacyEtcdReaderRole(kubeSystemNamespace),
					legacyEtcdReaderRoleBinding(kubeSystemNamespace),
				).
				Build(),
		),
	}

	reconciler := &InstanaAgentReconciler{client: countingClient}

	for range 3 {
		reconciler.cleanupLegacyEtcdReaderRBACOnce(context.Background(), agent, zap.New())
	}

	assertions.Equal(
		2,
		countingClient.deletes,
		"the Role and the RoleBinding should each be deleted exactly once",
	)
}

// A delete that fails for any reason other than the object being absent has to be retried,
// otherwise a single blip leaves the objects behind until the operator restarts.
func TestCleanupLegacyEtcdReaderRBACOnceRetriesAfterAFailedDelete(t *testing.T) {
	assertions := require.New(t)

	scheme := runtime.NewScheme()
	assertions.NoError(rbacv1.AddToScheme(scheme))

	agent := legacyEtcdReaderAgent()

	failingClient := &deleteCountingClient{
		InstanaAgentClient: instanaclient.NewInstanaAgentClient(
			fake.NewClientBuilder().WithScheme(scheme).Build(),
		),
		err: errors.New("apiserver temporary failure"),
	}

	reconciler := &InstanaAgentReconciler{client: failingClient}

	for range 3 {
		reconciler.cleanupLegacyEtcdReaderRBACOnce(context.Background(), agent, zap.New())
	}

	assertions.Equal(
		6,
		failingClient.deletes,
		"both deletes should be retried on every reconcile until they succeed",
	)
}

// Only the copies in kube-system were ever created by the operator, so identically named
// objects elsewhere must be left alone.
func TestCleanupLegacyEtcdReaderRBACLeavesOtherNamespacesAlone(t *testing.T) {
	assertions := require.New(t)

	scheme := runtime.NewScheme()
	assertions.NoError(rbacv1.AddToScheme(scheme))

	agent := legacyEtcdReaderAgent()

	baseClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(
			legacyEtcdReaderRole(agent.Namespace),
			legacyEtcdReaderRoleBinding(agent.Namespace),
		).
		Build()

	cleanupLegacyEtcdReaderRBAC(
		context.Background(),
		instanaclient.NewInstanaAgentClient(baseClient),
		agent,
		zap.New(),
	)

	key := types.NamespacedName{Name: legacyEtcdReaderName, Namespace: agent.Namespace}

	assertions.NoError(baseClient.Get(context.Background(), key, &rbacv1.Role{}))
	assertions.NoError(baseClient.Get(context.Background(), key, &rbacv1.RoleBinding{}))
}
