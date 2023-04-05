package operator_utils

import (
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/instana/instana-agent-operator/pkg/k8s/client"
	"github.com/instana/instana-agent-operator/pkg/result"
)

type OperatorUtils interface {
	ClusterIsOpenShift() result.Result[bool]
}

type operatorUtils struct {
	ctx context.Context
	client.InstanaAgentClient
}

func NewOperatorUtils(ctx context.Context, client client.InstanaAgentClient) OperatorUtils {
	return &operatorUtils{
		ctx:                ctx,
		InstanaAgentClient: client,
	}
}

// TODO: test

func (o *operatorUtils) crdIsInstalled(name string) result.Result[bool] {
	crdResult := o.GetAsResult(
		o.ctx, types.NamespacedName{Name: name}, &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "apiextensions.k8s.io/v1",
				"kind":       "CustomResourceDefinition",
			},
		},
	)
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

// TODO: test

func (o *operatorUtils) ClusterIsOpenShift() result.Result[bool] {
	return o.crdIsInstalled("clusteroperators.config.openshift.io")
}
