/*
(c) Copyright IBM Corp. 2025

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

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	v1 "github.com/instana/instana-agent-operator/api/v1"
)

func (r *RemoteAgentReconciler) addOrUpdateFinalizers(ctx context.Context, agentOld *v1.RemoteAgent) reconcileReturn {
	agentNew := agentOld.DeepCopy()

	if controllerutil.AddFinalizer(agentNew, finalizerV3) {
		r.loggerFor(ctx, agentNew).V(1).Info("adding agent finalizer to agent CR")
		return r.updateAgent(ctx, agentOld, agentNew)
	}

	return reconcileContinue()
}
