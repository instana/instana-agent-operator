/*
(c) Copyright IBM Corp. 2024
(c) Copyright Instana Inc.

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

package operator_utils

import (
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/collections/list"
	"github.com/instana/instana-agent-operator/pkg/k8s/client"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/transformations"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/lifecycle"
	"github.com/instana/instana-agent-operator/pkg/multierror"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

type OperatorUtils interface {
	ClusterIsOpenShift() (bool, error)
	ApplyAll(builders ...builder.ObjectBuilder) error
	DeleteAll() error
}

type operatorUtils struct {
	ctx                       context.Context
	builderTransformer        builder.BuilderTransformer
	dependentLifecycleManager lifecycle.DependentLifecycleManager
	instanaAgentClient        client.InstanaAgentClient
	instanaAgent              *instanav1.InstanaAgent
}

func NewOperatorUtils(
	ctx context.Context,
	instanaAgentClient client.InstanaAgentClient,
	agent *instanav1.InstanaAgent,
	dependentLifecycleManager lifecycle.DependentLifecycleManager,
) OperatorUtils {
	return &operatorUtils{
		ctx:                       ctx,
		builderTransformer:        builder.NewBuilderTransformer(transformations.NewTransformations(agent)),
		dependentLifecycleManager: dependentLifecycleManager,
		instanaAgentClient:        instanaAgentClient,
		instanaAgent:              agent,
	}
}

func (o *operatorUtils) ApplyAll(builders ...builder.ObjectBuilder) error {
	if err := o.applyAll(o.buildObjects(builders...), k8sclient.DryRunAll); err != nil {
		return err
	}

	objects := o.buildObjects(builders...)

	if err := o.dependentLifecycleManager.UpdateDependentLifecycleInfo(objects); err != nil {
		return err
	}

	if err := o.applyAll(objects); err != nil {
		return err
	}

	return o.dependentLifecycleManager.CleanupDependents(objects...)
}

func (o *operatorUtils) DeleteAll() error {
	return o.dependentLifecycleManager.CleanupDependents()
}

// ClusterIsOpenShift returns a boolean if the cluster has been defined as OpenShift
func (o *operatorUtils) ClusterIsOpenShift() (bool, error) {
	if o.instanaAgent.Spec.OpenShift == nil {
		return o.instanaAgentClient.Exists(
			o.ctx,
			schema.GroupVersionKind{
				Group:   "apiextensions.k8s.io",
				Version: "v1",
				Kind:    "CustomResourceDefinition",
			},
			types.NamespacedName{
				Name: "clusteroperators.config.openshift.io",
			},
		).Get()
	}

	return *o.instanaAgent.Spec.OpenShift, nil
}

func (o *operatorUtils) applyAll(objects []k8sclient.Object, opts ...k8sclient.PatchOption) error {
	errBuilder := multierror.NewMultiErrorBuilder()

	for _, obj := range objects {
		o.instanaAgentClient.Apply(o.ctx, obj, opts...).OnFailure(errBuilder.AddSingle)
	}

	return errBuilder.Build()
}

func (o *operatorUtils) buildObjects(builders ...builder.ObjectBuilder) []k8sclient.Object {
	optionals := list.
		NewListMapTo[builder.ObjectBuilder, optional.Optional[k8sclient.Object]]().
		MapTo(
			builders,
			o.builderTransformer.Apply,
		)

	return optional.
		NewNonEmptyOptionalMapper[k8sclient.Object]().
		AllNonEmpty(optionals)
}
