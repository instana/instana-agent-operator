package controllers

import (
	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	agentconfigmap "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/configmap"
	agentdaemonset "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/daemonset"
	headlessservice "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/headless-service"
	containersinstanaiosecret "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/secrets/containers-instana-io-secret"
	keyssecret "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/secrets/keys-secret"
	tlssecret "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/secrets/tls-secret"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/service"
	agentserviceaccount "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/serviceaccount"
	k8ssensorconfigmap "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/k8s-sensor/configmap"
	k8ssensorrbac "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/k8s-sensor/rbac"
	k8ssensorserviceaccount "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/k8s-sensor/serviceaccount"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/operator_utils"
)

func (r *InstanaAgentReconciler) applyResources(
	agent *instanav1.InstanaAgent,
	isOpenShift bool,
	operatorUtils operator_utils.OperatorUtils,
) reconcileReturn {
	switch res := operatorUtils.ApplyAll(
		agentconfigmap.NewConfigMapBuilder(agent),
		agentdaemonset.NewDaemonSetBuilder(agent, isOpenShift),
		headlessservice.NewHeadlessServiceBuilder(agent),
		containersinstanaiosecret.NewSecretBuilder(agent),
		keyssecret.NewSecretBuilder(agent),
		tlssecret.NewSecretBuilder(agent),
		service.NewServiceBuilder(agent),
		agentserviceaccount.NewServiceAccountBuilder(agent),
		k8ssensorconfigmap.NewConfigMapBuilder(agent),
		// TODO: K8s sensor deployment
		k8ssensorrbac.NewClusterRoleBuilder(agent),
		k8ssensorrbac.NewClusterRoleBindingBuilder(agent),
		k8ssensorserviceaccount.NewServiceAccountBuilder(agent),
	); res.IsSuccess() {
	case true:
		return reconcileContinue()
	default:
		_, err := res.Get()
		return reconcileFailure(err)
	}
}
