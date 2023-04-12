package operator_utils

import (
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/collections/list"
	"github.com/instana/instana-agent-operator/pkg/k8s/client"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/builder"
	"github.com/instana/instana-agent-operator/pkg/multierror"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/result"
)

type OperatorUtils interface {
	ClusterIsOpenShift() result.Result[bool]
	ApplyAll(builders []builder.ObjectBuilder) result.Result[[]k8sclient.Object]
	// TODO: Delete cluster-scoped for finalizer logic
	// TODO: delete previous generation leftovers -> behavior of namespace restriction for cluster-scoped resources?
}

type operatorUtils struct {
	ctx context.Context
	client.InstanaAgentClient
	*instanav1.InstanaAgent
	builderTransformer builder.BuilderTransformer
}

func NewOperatorUtils(
	ctx context.Context, client client.InstanaAgentClient, agent *instanav1.InstanaAgent,
) OperatorUtils {
	return &operatorUtils{
		ctx:                ctx,
		InstanaAgentClient: client,
		InstanaAgent:       agent,
		builderTransformer: builder.NewBuilderTransformer(agent),
	}
}

func (o *operatorUtils) crdIsInstalled(name string) result.Result[bool] {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("apiextensions.k8s.io/v1")
	obj.SetKind("CustomResourceDefinition")

	crdResult := o.GetAsResult(o.ctx, types.NamespacedName{Name: name}, obj)

	return result.Map(
		crdResult, func(_ k8sclient.Object) result.Result[bool] {
			return result.OfSuccess(true)
		},
	).Recover(
		func(err error) (bool, error) {
			return false, k8sclient.IgnoreNotFound(err)
		},
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

func (o *operatorUtils) applyAll(
	objects []k8sclient.Object, opts ...k8sclient.PatchOption,
) result.Result[[]k8sclient.Object] {
	errBuilder := multierror.NewMultiErrorBuilder()

	for _, obj := range objects {
		o.Apply(o.ctx, obj, opts...).
			OnFailure(
				func(err error) {
					errBuilder.Add(err)
				},
			)
	}

	return result.Of(objects, errBuilder.Build())
}

func (o *operatorUtils) ApplyAll(builders []builder.ObjectBuilder) result.Result[[]k8sclient.Object] {
	optionals := list.NewListMapTo[builder.ObjectBuilder, optional.Optional[k8sclient.Object]]().MapTo(
		builders,
		func(builder builder.ObjectBuilder) optional.Optional[k8sclient.Object] {
			return o.builderTransformer.Apply(builder)
		},
	)

	objects := optional.NewNonEmptyOptionalMapper[k8sclient.Object]().AllNonEmpty(optionals)

	switch res := o.applyAll(objects, k8sclient.DryRunAll); res.IsSuccess() {
	case true:
		return o.applyAll(objects)
	default:
		return res
	}
}
