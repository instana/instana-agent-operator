package controllers

import (
	"context"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	agentconfigmap "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/configmap"
	agentdaemonset "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/daemonset"
	headlessservice "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/headless-service"
	containersinstanaiosecret "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/secrets/containers-instana-io-secret"
	keyssecret "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/secrets/keys-secret"
	tlssecret "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/secrets/tls-secret"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/service"
	agentserviceaccount "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/serviceaccount"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	k8ssensorconfigmap "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/k8s-sensor/configmap"
	k8ssensordeployment "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/k8s-sensor/deployment"
	k8ssensorpoddisruptionbudget "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/k8s-sensor/poddisruptionbudget"
	k8ssensorrbac "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/k8s-sensor/rbac"
	k8ssensorserviceaccount "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/k8s-sensor/serviceaccount"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/operator_utils"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/status"
)

func getDaemonSetBuilders(
	agent *instanav1.InstanaAgent,
	isOpenShift bool,
	statusManager status.AgentStatusManager,
) []builder.ObjectBuilder {
	if len(agent.Spec.Zones) == 0 {
		return []builder.ObjectBuilder{agentdaemonset.NewDaemonSetBuilder(agent, isOpenShift, statusManager)}
	}

	builders := make([]builder.ObjectBuilder, 0, len(agent.Spec.Zones))

	for _, zone := range agent.Spec.Zones {
		builders = append(
			builders,
			agentdaemonset.NewDaemonSetBuilderWithZoneInfo(agent, isOpenShift, statusManager, &zone),
		)
	}

	return builders
}

func (r *InstanaAgentReconciler) applyResources(
	ctx context.Context,
	agent *instanav1.InstanaAgent,
	isOpenShift bool,
	operatorUtils operator_utils.OperatorUtils,
	statusManager status.AgentStatusManager,
) reconcileReturn {
	log := r.loggerFor(ctx, agent)
	log.V(1).Info("applying Kubernetes resources for agent")

	builders := append(
		getDaemonSetBuilders(agent, isOpenShift, statusManager),
		agentconfigmap.NewConfigMapBuilder(agent, statusManager),
		headlessservice.NewHeadlessServiceBuilder(agent),
		containersinstanaiosecret.NewSecretBuilder(agent),
		keyssecret.NewSecretBuilder(agent),
		tlssecret.NewSecretBuilder(agent),
		service.NewServiceBuilder(agent),
		agentserviceaccount.NewServiceAccountBuilder(agent),
		k8ssensorconfigmap.NewConfigMapBuilder(agent),
		k8ssensordeployment.NewDeploymentBuilder(agent, isOpenShift, statusManager),
		k8ssensorpoddisruptionbudget.NewPodDisruptionBudgetBuilder(agent),
		k8ssensorrbac.NewClusterRoleBuilder(agent),
		k8ssensorrbac.NewClusterRoleBindingBuilder(agent),
		k8ssensorserviceaccount.NewServiceAccountBuilder(agent),
	)

	switch res := operatorUtils.ApplyAll(builders...); res.IsSuccess() {
	case true:
		log.V(1).Info("successfully applied kubernetes resources for agent")
		return reconcileContinue()
	default:
		_, err := res.Get()
		log.Error(err, "failed to apply kubernetes resources for agent")
		return reconcileFailure(err)
	}
}
