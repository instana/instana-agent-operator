/*
(c) Copyright IBM Corp. 2024, 2025

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
	"sort"
	"strings"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/client"
	namespaces_configmap "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/configmap/namespaces-configmap"
	agentdaemonset "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/daemonset"
	headlessservice "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/headless-service"
	agentrbac "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/rbac"
	agentsecrets "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/secrets"
	keyssecret "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/secrets/keys-secret"
	tlssecret "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/secrets/tls-secret"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/service"
	agentserviceaccount "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/serviceaccount"
	backends "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/backends"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/namespaces"
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

func getK8sSensorDeployments(
	agent *instanav1.InstanaAgent,
	isOpenShift bool,
	statusManager status.AgentStatusManager,
	k8SensorBackends []backends.K8SensorBackend,
	keysSecret *corev1.Secret,
	deploymentContext *k8ssensordeployment.DeploymentContext,
) []builder.ObjectBuilder {
	builders := make([]builder.ObjectBuilder, 0, len(k8SensorBackends))

	for _, backend := range k8SensorBackends {
		builders = append(
			builders,
			k8ssensordeployment.NewDeploymentBuilder(agent, isOpenShift, statusManager, backend, keysSecret, deploymentContext),
		)
	}

	return builders
}

// getSortedTargets returns a sorted copy of the given targets slice
func getSortedTargets(targets []string) []string {
	sortedTargets := make([]string, len(targets))
	copy(sortedTargets, targets)
	sort.Strings(sortedTargets)
	return sortedTargets
}

// compareAndUpdateETCDTargets compares current ETCD targets with new discovered targets
// and determines if an update is needed. This function handles environment variable extraction,
// target sorting, and comparison logic.
func compareAndUpdateETCDTargets(
	existingDeployment *appsv1.Deployment,
	discoveredTargets []string,
	log logr.Logger,
) bool {
	log.Info(
		"Comparing current ETCD targets with discovered targets to determine if update is needed",
	)

	// Extract current ETCD targets from deployment environment variables
	currentTargets := ""
	for _, container := range existingDeployment.Spec.Template.Spec.Containers {
		if container.Name == constants.ContainerK8Sensor {
			for _, env := range container.Env {
				if env.Name == constants.EnvETCDTargets {
					currentTargets = env.Value
					break
				}
			}
			break
		}
	}

	// Sort discovered targets to ensure consistent comparison
	sortedDiscoveredTargets := getSortedTargets(discoveredTargets)
	newTargets := strings.Join(sortedDiscoveredTargets, ",")

	// Sort current targets for proper comparison
	if currentTargets != "" {
		currentTargetsList := strings.Split(currentTargets, ",")
		sort.Strings(currentTargetsList)
		currentTargets = strings.Join(currentTargetsList, ",")
	}

	log.Info("Target comparison details",
		"currentTargets", currentTargets,
		"newTargets", newTargets,
		"needsUpdate", currentTargets != newTargets)

	// Return true if targets are different (update needed)
	return currentTargets != newTargets
}

// ETCDDiscoverFunc is a function type for discovering ETCD endpoints
type ETCDDiscoverFunc func(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error)

// CreateDeploymentContext creates a deployment context for the k8s-sensor deployment.
// It handles both OpenShift and vanilla Kubernetes cases, setting up the appropriate
// ETCD configuration based on the environment.
func CreateDeploymentContext(
	ctx context.Context,
	c client.InstanaAgentClient,
	agent *instanav1.InstanaAgent,
	isOpenShift bool,
	logger logr.Logger,
	discoverETCD ETCDDiscoverFunc,
) (*k8ssensordeployment.DeploymentContext, error) {
	var deploymentContext *k8ssensordeployment.DeploymentContext

	// For OpenShift, create the service-CA ConfigMap
	if isOpenShift {
		if err := CreateServiceCAConfigMap(ctx, c, agent, logger); err != nil {
			logger.Error(err, "Failed to create service-CA ConfigMap")
			// Continue with reconciliation, don't fail the whole process
			return deploymentContext, nil
		}

		// Set up deployment context for OpenShift
		deploymentContext = &k8ssensordeployment.DeploymentContext{
			ETCDCASecretName: constants.ServiceCAConfigMapName,
		}
		return deploymentContext, nil
	}

	// For vanilla Kubernetes, discover ETCD endpoints
	discoveredETCD, err := discoverETCD(ctx, agent)
	if err != nil {
		logger.Error(err, "Failed to discover ETCD endpoints")
		// Continue with reconciliation, don't fail the whole process
		return deploymentContext, nil
	}

	if discoveredETCD == nil || len(discoveredETCD.Targets) == 0 {
		return deploymentContext, nil
	}

	// Check if we need to update the Deployment with new ETCD targets
	existingDeployment := &appsv1.Deployment{}
	helperInstance := helpers.NewHelpers(agent)
	err = c.Get(ctx, types.NamespacedName{
		Namespace: agent.Namespace,
		Name:      helperInstance.K8sSensorResourcesName(),
	}, existingDeployment)

	if err != nil {
		logger.Info("K8sensor deployment not found, will create with discovered ETCD targets")
	} else {
		// Compare current and discovered ETCD targets
		needsUpdate := compareAndUpdateETCDTargets(existingDeployment, discoveredETCD.Targets, logger)
		if !needsUpdate {
			logger.Info("ETCD targets unchanged, skipping Deployment update")
			return nil, nil
		}
	}

	// Use sorted targets for consistency
	sortedTargets := getSortedTargets(discoveredETCD.Targets)

	logger.Info("Using discovered ETCD targets", "targets", sortedTargets)
	deploymentContext = &k8ssensordeployment.DeploymentContext{
		DiscoveredETCDTargets: sortedTargets,
	}
	if discoveredETCD.CAFound {
		deploymentContext.ETCDCASecretName = constants.ETCDCASecretName
	}

	return deploymentContext, nil
}

// createDeploymentContext creates a deployment context for the k8s-sensor deployment.
// It handles both OpenShift and vanilla Kubernetes cases, setting up the appropriate
// ETCD configuration based on the environment.
func (r *InstanaAgentReconciler) createDeploymentContext(
	ctx context.Context,
	agent *instanav1.InstanaAgent,
	isOpenShift bool,
) (*k8ssensordeployment.DeploymentContext, reconcileReturn) {
	log := r.loggerFor(ctx, agent)

	deploymentContext, err := CreateDeploymentContext(ctx, r.client, agent, isOpenShift, log, r.DiscoverETCDEndpoints)
	if err != nil {
		return nil, reconcileFailure(err)
	}

	// Handle the special case where targets are unchanged
	if !isOpenShift && deploymentContext == nil {
		return nil, reconcileContinue()
	}

	return deploymentContext, reconcileContinue()
}

func (r *InstanaAgentReconciler) applyResources(
	ctx context.Context,
	agent *instanav1.InstanaAgent,
	isOpenShift bool,
	operatorUtils operator_utils.OperatorUtils,
	statusManager status.AgentStatusManager,
	keysSecret *corev1.Secret,
	k8SensorBackends []backends.K8SensorBackend,
	namespacesDetails namespaces.NamespacesDetails,
) reconcileReturn {
	log := r.loggerFor(ctx, agent)
	log.V(1).Info("applying Kubernetes resources for agent")

	// Create deployment context for k8s-sensor
	deploymentContext, result := r.createDeploymentContext(ctx, agent, isOpenShift)
	if result.suppliesReconcileResult() {
		return result
	}

	builders := append(
		getDaemonSetBuilders(agent, isOpenShift, statusManager),
		headlessservice.NewHeadlessServiceBuilder(agent),
		agentsecrets.NewConfigBuilder(agent, statusManager, keysSecret, k8SensorBackends),
		agentsecrets.NewContainerBuilder(agent, keysSecret),
		tlssecret.NewSecretBuilder(agent),
		service.NewServiceBuilder(agent),
		agentrbac.NewClusterRoleBuilder(agent),
		agentrbac.NewClusterRoleBindingBuilder(agent),
		agentserviceaccount.NewServiceAccountBuilder(agent),
		k8ssensorpoddisruptionbudget.NewPodDisruptionBudgetBuilder(agent),
		k8ssensorrbac.NewClusterRoleBuilder(agent),
		k8ssensorrbac.NewClusterRoleBindingBuilder(agent),
		k8ssensorrbac.NewRoleBuilder(agent),
		k8ssensorrbac.NewRoleBindingBuilder(agent),
		k8ssensorserviceaccount.NewServiceAccountBuilder(agent),
		k8ssensorconfigmap.NewConfigMapBuilder(agent, k8SensorBackends),
		keyssecret.NewSecretBuilder(agent, k8SensorBackends),
		namespaces_configmap.NewConfigMapBuilder(agent, statusManager, namespacesDetails),
	)

	builders = append(builders, getK8sSensorDeployments(agent, isOpenShift, statusManager, k8SensorBackends, keysSecret, deploymentContext)...)

	if err := operatorUtils.ApplyAll(builders...); err != nil {
		log.Error(err, "failed to apply kubernetes resources for agent")
		return reconcileFailure(err)
	}

	log.V(1).Info("successfully applied kubernetes resources for agent")
	return reconcileContinue()
}
