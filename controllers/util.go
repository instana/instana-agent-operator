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

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/operator_utils"
)

func (r *InstanaAgentReconciler) isOpenShift(ctx context.Context, operatorUtils operator_utils.OperatorUtils) (
	bool,
	reconcileReturn,
) {
	log := logf.FromContext(ctx)

	isOpenShiftRes, err := operatorUtils.ClusterIsOpenShift()
	if err != nil {
		log.Error(err, "failed to determine if cluster is OpenShift")
		return false, reconcileFailure(err)
	}
	log.V(1).Info("successfully detected whether cluster is OpenShift", "IsOpenShift", isOpenShiftRes)
	return isOpenShiftRes, reconcileContinue()
}

func (r *InstanaAgentReconciler) loggerFor(ctx context.Context, agent *instanav1.InstanaAgent) logr.Logger {
	return logf.FromContext(ctx).WithValues(
		"Generation",
		agent.Generation,
		"UID",
		agent.UID,
	)
}
