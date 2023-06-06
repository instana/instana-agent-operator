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
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/lifecycle"
	"github.com/instana/instana-agent-operator/pkg/multierror"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/result"
)

type OperatorUtils interface {
	ClusterIsOpenShift() result.Result[bool]
	ApplyAll(builders ...builder.ObjectBuilder) result.Result[[]k8sclient.Object]
	DeleteAll() result.Result[[]k8sclient.Object]
}

type operatorUtils struct {
	ctx context.Context
	client.InstanaAgentClient
	*instanav1.InstanaAgent
	builderTransformer builder.BuilderTransformer
	lifecycle.DependentLifecycleManager
}

func NewOperatorUtils(
	ctx context.Context, client client.InstanaAgentClient, agent *instanav1.InstanaAgent,
) OperatorUtils {
	return &operatorUtils{
		ctx:                       ctx,
		InstanaAgentClient:        client,
		InstanaAgent:              agent,
		builderTransformer:        builder.NewBuilderTransformer(agent),
		DependentLifecycleManager: lifecycle.NewDependentLifecycleManager(ctx, agent, client),
	}
}

func (o *operatorUtils) crdIsInstalled(name string) result.Result[bool] {
	return o.Exists(
		o.ctx, schema.GroupVersionKind{
			Group:   "apiextensions.k8s.io",
			Version: "v1",
			Kind:    "CustomResourceDefinition",
		}, types.NamespacedName{Name: name},
	)

}

func (o *operatorUtils) ClusterIsOpenShift() result.Result[bool] {
	switch userProvided := o.Spec.OpenShift; userProvided {
	case nil:
		return o.crdIsInstalled("clusteroperators.config.openshift.io")
	default:
		return result.OfSuccess(*userProvided)
	}
}

func (o *operatorUtils) applyAllWithOpts(
	objects []k8sclient.Object, opts ...k8sclient.PatchOption,
) result.Result[[]k8sclient.Object] {
	errBuilder := multierror.NewMultiErrorBuilder()

	for _, obj := range objects {
		o.Apply(o.ctx, obj, opts...).OnFailure(errBuilder.AddSingle)
	}

	return result.Of(objects, errBuilder.Build())
}

func (o *operatorUtils) applyAll(objects []k8sclient.Object) result.Result[[]k8sclient.Object] {
	return o.applyAllWithOpts(objects)
}

func (o *operatorUtils) applyAllDryRun(objects []k8sclient.Object) result.Result[[]k8sclient.Object] {
	return o.applyAllWithOpts(objects, k8sclient.DryRunAll)
}

func (o *operatorUtils) ApplyAll(builders ...builder.ObjectBuilder) result.Result[[]k8sclient.Object] {
	optionals := list.NewListMapTo[builder.ObjectBuilder, optional.Optional[k8sclient.Object]]().MapTo(
		builders,
		o.builderTransformer.Apply,
	)

	objects := optional.NewNonEmptyOptionalMapper[k8sclient.Object]().AllNonEmpty(optionals)

	dryRunRes := o.applyAllDryRun(objects)

	updateLifecycleCmRes := result.Map[[]k8sclient.Object, []k8sclient.Object](
		dryRunRes,
		o.UpdateDependentLifecycleInfo,
	)

	applyRes := result.Map[[]k8sclient.Object, []k8sclient.Object](updateLifecycleCmRes, o.applyAll)

	return result.Map[[]k8sclient.Object, []k8sclient.Object](
		applyRes,
		o.DeleteOrphanedDependents,
	)
}

func (o *operatorUtils) DeleteAll() result.Result[[]k8sclient.Object] {
	return o.DeleteAllDependents()
}
