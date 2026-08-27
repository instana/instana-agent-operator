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

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	instanaclient "github.com/instana/instana-agent-operator/pkg/k8s/client"
)

type getErrorInstanaAgentClient struct {
	instanaclient.InstanaAgentClient
	err error
}

func (c *getErrorInstanaAgentClient) Get(
	ctx context.Context,
	key types.NamespacedName,
	obj client.Object,
	opts ...client.GetOption,
) error {
	return c.err
}

func TestShouldSetPersistHostUniqueIDEnvVarWhenDaemonSetNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	assert.NoError(t, appsv1.AddToScheme(scheme))

	reconciler := &InstanaAgentReconciler{
		client: instanaclient.NewInstanaAgentClient(
			fake.NewClientBuilder().WithScheme(scheme).Build(),
		),
	}
	agent := &instanav1.InstanaAgent{}
	agent.Name = "test-agent"
	agent.Namespace = "test-namespace"

	shouldSet, res := reconciler.shouldSetPersistHostUniqueIDEnvVar(
		context.Background(),
		agent,
		nil,
	)
	assert.True(t, shouldSet)
	assert.False(t, res.suppliesReconcileResult())
}

func TestShouldSetPersistHostUniqueIDEnvVarGetErrorReturnsFailure(t *testing.T) {
	scheme := runtime.NewScheme()
	assert.NoError(t, appsv1.AddToScheme(scheme))

	getErr := errors.New("temporary apiserver outage")
	baseClient := instanaclient.NewInstanaAgentClient(
		fake.NewClientBuilder().WithScheme(scheme).Build(),
	)

	reconciler := &InstanaAgentReconciler{
		client: &getErrorInstanaAgentClient{
			InstanaAgentClient: baseClient,
			err:                getErr,
		},
	}
	agent := &instanav1.InstanaAgent{}
	agent.Name = "test-agent"
	agent.Namespace = "test-namespace"

	shouldSet, res := reconciler.shouldSetPersistHostUniqueIDEnvVar(
		context.Background(),
		agent,
		nil,
	)

	assert.False(t, shouldSet)
	assert.True(t, res.suppliesReconcileResult())
	_, err := res.reconcileResult()
	assert.ErrorIs(t, err, getErr)
}
