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
	"fmt"
	"sort"
	"strings"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		return []builder.ObjectBuilder{
			agentdaemonset.NewDaemonSetBuilder(agent, isOpenShift, statusManager),
		}
	}

	builders := make([]builder.ObjectBuilder, 0, len(agent.Spec.Zones))

	for _, zone := range agent.Spec.Zones {
		builders = append(
			builders,
			agentdaemonset.NewDaemonSetBuilderWithZoneInfo(
				agent,
				isOpenShift,
				statusManager,
				&zone,
			),
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
			k8ssensordeployment.NewDeploymentBuilder(
				agent,
				isOpenShift,
				statusManager,
				backend,
				keysSecret,
				deploymentContext,
			),
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

// validateOpenShiftETCDResources validates that the ETCD ConfigMap and Secret contain required keys
func validateOpenShiftETCDResources(
	configMap *corev1.ConfigMap,
	secret *corev1.Secret,
	caErr, certErr error,
	logger logr.Logger,
) (error, error) {
	// If CA already has error, skip validation
	if caErr != nil {
		return caErr, certErr
	}

	// Validate ConfigMap contains required keys
	if configMap.Data == nil {
		logger.Info(
			"OpenShift ETCD CA bundle ConfigMap has no data, ETCD monitoring will be disabled",
		)
		return fmt.Errorf("etcd-metrics-ca-bundle has no data"), certErr
	}

	if _, ok := configMap.Data["ca-bundle.crt"]; !ok {
		logger.Info(
			"OpenShift ETCD CA bundle missing ca-bundle.crt key, ETCD monitoring will be disabled",
		)
		return fmt.Errorf("etcd-metrics-ca-bundle missing ca-bundle.crt key"), certErr
	}

	if configMap.Data["ca-bundle.crt"] == "" {
		logger.Info(
			"OpenShift ETCD CA bundle ca-bundle.crt is empty, ETCD monitoring will be disabled",
		)
		return fmt.Errorf("etcd-metrics-ca-bundle ca-bundle.crt is empty"), certErr
	}

	// If Secret already has error, skip validation
	if certErr != nil {
		return caErr, certErr
	}

	// Validate Secret contains required keys
	if secret.Data == nil {
		logger.Info(
			"OpenShift ETCD client Secret has no data, ETCD monitoring will be disabled",
		)
		return caErr, fmt.Errorf("etcd-metric-client has no data")
	}

	// Check for tls.crt
	if _, ok := secret.Data["tls.crt"]; !ok {
		logger.Info(
			"OpenShift ETCD client Secret missing tls.crt key, ETCD monitoring will be disabled",
		)
		return caErr, fmt.Errorf("etcd-metric-client missing tls.crt key")
	}

	if len(secret.Data["tls.crt"]) == 0 {
		logger.Info(
			"OpenShift ETCD client Secret tls.crt is empty, ETCD monitoring will be disabled",
		)
		return caErr, fmt.Errorf("etcd-metric-client tls.crt is empty")
	}

	// Check for tls.key
	if _, ok := secret.Data["tls.key"]; !ok {
		logger.Info(
			"OpenShift ETCD client Secret missing tls.key key, ETCD monitoring will be disabled",
		)
		return caErr, fmt.Errorf("etcd-metric-client missing tls.key key")
	}

	if len(secret.Data["tls.key"]) == 0 {
		logger.Info(
			"OpenShift ETCD client Secret tls.key is empty, ETCD monitoring will be disabled",
		)
		return caErr, fmt.Errorf("etcd-metric-client tls.key is empty")
	}

	return caErr, certErr
}

// buildETCDResourceObjectMeta creates ObjectMeta for copied ETCD resources with
// owner references, labels, and synchronization tracking annotations
func buildETCDResourceObjectMeta(
	targetName string,
	sourceName string,
	sourceResourceVersion string,
	agent *instanav1.InstanaAgent,
) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      targetName,
		Namespace: agent.Namespace,
		OwnerReferences: []metav1.OwnerReference{
			*metav1.NewControllerRef(agent, instanav1.GroupVersion.WithKind("InstanaAgent")),
		},
		Labels: map[string]string{
			"app.kubernetes.io/name":       "instana-agent",
			"app.kubernetes.io/component":  "k8sensor",
			"app.kubernetes.io/managed-by": "instana-agent-operator",
			"instana.io/copied-from":       constants.ETCDNamespace,
		},
		Annotations: map[string]string{
			"instana.io/source-namespace":        constants.ETCDNamespace,
			"instana.io/source-name":             sourceName,
			"instana.io/source-resource-version": sourceResourceVersion,
			"instana.io/instana-agent-name":      agent.Name,
		},
	}
}

// copyETCDResourcesToNamespace copies ETCD ConfigMap and Secret to the target namespace
func copyETCDResourcesToNamespace(
	ctx context.Context,
	c client.InstanaAgentClient,
	agent *instanav1.InstanaAgent,
	sourceConfigMap *corev1.ConfigMap,
	sourceSecret *corev1.Secret,
	logger logr.Logger,
) bool {
	// Copy ETCD ConfigMap to instana-agent namespace with synchronization tracking
	targetCAConfigMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: buildETCDResourceObjectMeta(
			constants.ETCDMetricsCABundleName,
			constants.ETCDMetricsCABundleName,
			sourceConfigMap.ResourceVersion,
			agent,
		),
		Data: sourceConfigMap.Data,
	}
	if _, err := c.Apply(ctx, targetCAConfigMap).Get(); err != nil {
		logger.Error(err, "Failed to copy ETCD CA ConfigMap to instana-agent namespace")
		return false
	}
	logger.Info("Successfully copied/updated ETCD CA ConfigMap",
		"sourceResourceVersion", sourceConfigMap.ResourceVersion)

	// Copy ETCD Secret to instana-agent namespace with synchronization tracking
	targetClientSecret := &corev1.Secret{ // pragma: allowlist secret
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: buildETCDResourceObjectMeta(
			constants.ETCDMetricClientSecretName,
			constants.ETCDMetricClientSecretName,
			sourceSecret.ResourceVersion,
			agent,
		),
		Type: sourceSecret.Type,
		Data: sourceSecret.Data,
	}
	if _, err := c.Apply(ctx, targetClientSecret).Get(); err != nil {
		logger.Error(err, "Failed to copy ETCD client Secret to instana-agent namespace")

		// NOTE: If ConfigMap copy succeeded but Secret copy fails, the ConfigMap
		// will remain in the namespace temporarily until the next reconciliation loop,
		// where the cleanup logic will remove it. This is acceptable because:
		// 1. OpenShiftETCDResourcesExist=false prevents mounting incomplete resources
		// 2. The operator reconciles frequently (default: every few minutes)
		// 3. The ConfigMap is harmless without the Secret
		return false
	}
	logger.Info("Successfully copied/updated ETCD client Secret",
		"sourceResourceVersion", sourceSecret.ResourceVersion)

	return true
}

// cleanupCopiedETCDResources removes copied ETCD resources from the target namespace
func cleanupCopiedETCDResources(
	ctx context.Context,
	c client.InstanaAgentClient,
	agent *instanav1.InstanaAgent,
	logger logr.Logger,
) {
	// Clean up copied ConfigMap if it exists
	copiedConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.ETCDMetricsCABundleName,
			Namespace: agent.Namespace,
		},
	}
	if err := c.Delete(ctx, copiedConfigMap); err != nil && !apierrors.IsNotFound(err) {
		logger.Error(err, "Failed to cleanup copied ETCD CA ConfigMap")
	} else if err == nil {
		logger.Info("Cleaned up copied ETCD CA ConfigMap")
	}

	// Clean up copied Secret if it exists
	copiedSecret := &corev1.Secret{ // pragma: allowlist secret
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.ETCDMetricClientSecretName,
			Namespace: agent.Namespace,
		},
	}
	if err := c.Delete(ctx, copiedSecret); err != nil && !apierrors.IsNotFound(err) {
		logger.Error(err, "Failed to cleanup copied ETCD client Secret")
	} else if err == nil {
		logger.Info("Cleaned up copied ETCD client Secret")
	}
}

// logAndCleanupETCDErrors logs ETCD resource errors and performs cleanup
func logAndCleanupETCDErrors(
	ctx context.Context,
	c client.InstanaAgentClient,
	agent *instanav1.InstanaAgent,
	caErr, certErr error,
	logger logr.Logger,
) {
	if caErr != nil {
		logger.Info(
			"OpenShift ETCD CA bundle not found, ETCD monitoring will be disabled",
			"error",
			caErr,
		)
	}
	if certErr != nil {
		logger.Info(
			"OpenShift ETCD client certificate not found, ETCD monitoring will be disabled",
			"error",
			certErr,
		)
	}

	cleanupCopiedETCDResources(ctx, c, agent, logger)
}

// setupOpenShiftETCDMonitoring sets up ETCD monitoring for OpenShift clusters
func setupOpenShiftETCDMonitoring(
	ctx context.Context,
	c client.InstanaAgentClient,
	agent *instanav1.InstanaAgent,
	logger logr.Logger,
) *k8ssensordeployment.DeploymentContext {
	deploymentContext := &k8ssensordeployment.DeploymentContext{}

	// Fetch ETCD resources from openshift-etcd namespace
	etcdCAConfigMap := &corev1.ConfigMap{}
	caErr := c.Get(ctx, types.NamespacedName{
		Namespace: constants.ETCDNamespace,
		Name:      constants.ETCDMetricsCABundleName,
	}, etcdCAConfigMap)

	etcdClientSecret := &corev1.Secret{} // pragma: allowlist secret
	certErr := c.Get(ctx, types.NamespacedName{
		Namespace: constants.ETCDNamespace,
		Name:      constants.ETCDMetricClientSecretName,
	}, etcdClientSecret)

	// Validate resources contain required keys
	caErr, certErr = validateOpenShiftETCDResources(
		etcdCAConfigMap,
		etcdClientSecret,
		caErr,
		certErr,
		logger,
	)

	// If either resource is invalid, log errors, cleanup, and return early
	if caErr != nil || certErr != nil {
		logAndCleanupETCDErrors(ctx, c, agent, caErr, certErr, logger)
		return deploymentContext
	}

	// Happy path - both resources are valid
	logger.Info("OpenShift ETCD resources found, enabling ETCD monitoring")

	// Copy resources to instana-agent namespace
	if copyETCDResourcesToNamespace(ctx, c, agent, etcdCAConfigMap, etcdClientSecret, logger) {
		deploymentContext.OpenShiftETCDResourcesExist = true
	}

	return deploymentContext
}

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
	if isOpenShift {
		return setupOpenShiftETCDMonitoring(ctx, c, agent, logger), nil
	}

	// For vanilla Kubernetes, discover ETCD endpoints
	discoveredETCD, err := discoverETCD(ctx, agent)
	if err != nil {
		logger.Error(err, "Failed to discover ETCD endpoints")
		// Continue with reconciliation, don't fail the whole process
		return nil, nil
	}

	if discoveredETCD == nil || len(discoveredETCD.Targets) == 0 {
		return nil, nil
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
	deploymentContext := &k8ssensordeployment.DeploymentContext{
		DiscoveredETCDTargets: sortedTargets,
	}
	if discoveredETCD.CAFound {
		deploymentContext.ETCDCASecretName = constants.ETCDCASecretName
	}

	return deploymentContext, nil
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
	deploymentContext, err := CreateDeploymentContext(
		ctx,
		r.client,
		agent,
		isOpenShift,
		log,
		r.DiscoverETCDEndpoints,
	)
	if err != nil {
		log.Error(err, "failed to create deployment context")
		return reconcileFailure(err)
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

	builders = append(
		builders,
		getK8sSensorDeployments(
			agent,
			isOpenShift,
			statusManager,
			k8SensorBackends,
			keysSecret,
			deploymentContext,
		)...)

	if err := operatorUtils.ApplyAll(builders...); err != nil {
		log.Error(err, "failed to apply kubernetes resources for agent")
		return reconcileFailure(err)
	}

	log.V(1).Info("successfully applied kubernetes resources for agent")
	return reconcileContinue()
}
