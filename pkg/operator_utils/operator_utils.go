package operator_utils

import (
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/client"
	"github.com/instana/instana-agent-operator/pkg/result"
)

type OperatorUtils interface {
	ClusterIsOpenShift() result.Result[bool]
}

type operatorUtils struct {
	ctx context.Context
	client.InstanaAgentClient
	*instanav1.InstanaAgent
}

func NewOperatorUtils(
	ctx context.Context,
	client client.InstanaAgentClient,
	agent *instanav1.InstanaAgent,
) OperatorUtils {
	return &operatorUtils{
		ctx:                ctx,
		InstanaAgentClient: client,
		InstanaAgent:       agent,
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
