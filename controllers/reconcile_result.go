package controllers

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/result"
)

type reconcileReturn struct {
	res optional.Optional[result.Result[ctrl.Result]]
}

func (r reconcileReturn) suppliesReconcileResult() bool {
	return r.res.IsNotEmpty()
}

func (r reconcileReturn) reconcileResult() (ctrl.Result, error) {
	return r.res.Get().Get()
}

func reconcileSuccess(res ctrl.Result) reconcileReturn {
	return reconcileReturn{optional.Of(result.OfSuccess(res))}
}

func reconcileFailure(err error) reconcileReturn {
	return reconcileReturn{optional.Of(result.OfFailure[ctrl.Result](err))}
}

func reconcileContinue() reconcileReturn {
	return reconcileReturn{optional.Empty[result.Result[ctrl.Result]]()}
}
