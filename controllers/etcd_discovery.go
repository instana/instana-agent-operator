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
	"fmt"
	"sort"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/operator_utils"
)

// DiscoveredETCDTargets holds information about discovered ETCD endpoints
type DiscoveredETCDTargets struct {
	// Targets is a slice of ETCD endpoint URLs in the format scheme://ip:port/metrics
	// where scheme is either http or https. For example: "https://10.0.0.1:2379/metrics"
	Targets []string

	// CAFound indicates whether the etcd-ca secret was found in the agent's namespace,
	// which is needed for secure HTTPS connections to etcd endpoints
	CAFound bool
}

const kubeSystemNamespace = "kube-system"

// DiscoverETCDEndpoints attempts to discover ETCD endpoints in vanilla Kubernetes clusters.
func (r *InstanaAgentReconciler) DiscoverETCDEndpoints(
	ctx context.Context,
	agent *instanav1.InstanaAgent,
) (*DiscoveredETCDTargets, error) {
	log := r.loggerFor(ctx, agent)

	// Step 1: Check if discovery should be skipped
	shouldSkip, err := r.shouldSkipDiscovery(ctx, agent)
	if err != nil {
		return nil, err
	}
	if shouldSkip {
		log.Info("Skipping ETCD discovery based on configuration or environment")
		return nil, nil
	}

	// Step 2: Find etcd service
	etcdService, err := r.findETCDService(ctx, log)
	if err != nil {
		return nil, err
	}
	if etcdService == nil {
		log.Info("No ETCD service found in kube-system namespace")
		return nil, nil
	}

	log.Info("Found etcd service", "name", etcdService.Name)

	// Step 3: Find metrics port and determine scheme
	metricsPortPtr, scheme := r.findMetricsPortAndScheme(etcdService)
	if metricsPortPtr == nil {
		log.Info("No metrics port found in etcd service")
		return nil, nil
	}
	metricsPort := *metricsPortPtr

	// Step 4: Get endpoints and build targets
	targets, err := r.buildTargetsFromEndpoints(ctx, etcdService, metricsPort, scheme)
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		log.Info("No endpoints found for etcd service")
		return nil, nil
	}

	// Step 5: Check for CA secret and return results
	caSecretExists := r.checkCASecretExists(ctx, agent)

	log.Info("Discovered etcd targets", "targets", targets, "caFound", caSecretExists)

	return &DiscoveredETCDTargets{
		Targets: targets,
		CAFound: caSecretExists,
	}, nil
}

// shouldSkipDiscovery checks if ETCD discovery should be skipped
func (r *InstanaAgentReconciler) shouldSkipDiscovery(
	ctx context.Context,
	agent *instanav1.InstanaAgent,
) (bool, error) {
	operatorUtils := operator_utils.NewOperatorUtils(ctx, r.client, agent, nil)
	isOpenShift, isOpenShiftRes := r.isOpenShift(ctx, operatorUtils)

	if isOpenShiftRes.suppliesReconcileResult() {
		return false, fmt.Errorf("failed to determine if cluster is OpenShift")
	}

	if isOpenShift {
		r.loggerFor(ctx, agent).Info("Skipping ETCD discovery on OpenShift cluster")
		return true, nil
	}

	if len(agent.Spec.K8sSensor.ETCD.Targets) > 0 {
		r.loggerFor(ctx, agent).
			Info("Using ETCD targets from CR spec", "targets", agent.Spec.K8sSensor.ETCD.Targets)
		return true, nil
	}

	return false, nil
}

// findETCDService attempts to find an etcd service in the kube-system namespace
func (r *InstanaAgentReconciler) findETCDService(
	ctx context.Context,
	log logr.Logger,
) (*corev1.Service, error) {
	// Try services with component=etcd label first
	if service, err := r.getServiceWithLabel(ctx, "etcd", "component", "etcd"); err != nil {
		return nil, err
	} else if service != nil {
		log.Info("Found etcd service with component=etcd label", "name", service.Name)
		return service, nil
	}

	if service, err := r.getServiceWithLabel(ctx, "etcd-metrics", "component", "etcd"); err != nil {
		return nil, err
	} else if service != nil {
		log.Info("Found etcd-metrics service with component=etcd label", "name", service.Name)
		return service, nil
	}

	// Fallback to name-based search
	log.Info("No service found with component=etcd label, trying by name")

	// Try by name in sequence
	serviceNames := []string{"etcd", "etcd-metrics", "etcd-k8s"}
	for _, name := range serviceNames {
		service, err := r.getServiceByName(ctx, name)
		if err != nil {
			return nil, err
		}
		if service != nil {
			return service, nil
		}
	}

	return nil, nil
}

// getServiceWithLabel gets a service with the specified name and checks if it has the expected label
func (r *InstanaAgentReconciler) getServiceWithLabel(
	ctx context.Context,
	name, labelKey, labelValue string,
) (*corev1.Service, error) {
	service := &corev1.Service{}
	err := r.client.Get(ctx, types.NamespacedName{
		Namespace: kubeSystemNamespace,
		Name:      name,
	}, service)

	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	if service.Labels == nil || service.Labels[labelKey] != labelValue {
		return nil, nil
	}

	return service, nil
}

// getServiceByName gets a service by name
func (r *InstanaAgentReconciler) getServiceByName(
	ctx context.Context,
	name string,
) (*corev1.Service, error) {
	service := &corev1.Service{}
	err := r.client.Get(ctx, types.NamespacedName{
		Namespace: kubeSystemNamespace,
		Name:      name,
	}, service)

	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return service, nil
}

// findMetricsPortAndScheme finds the metrics port and determines the scheme
func (r *InstanaAgentReconciler) findMetricsPortAndScheme(
	service *corev1.Service,
) (*int32, string) {
	for _, port := range service.Spec.Ports {
		if port.Name == "metrics" {
			// Use switch/case for scheme determination
			scheme := "https" // Default to https for unknown ports
			switch port.Port {
			case constants.ETCDMetricsPortHTTPS:
				scheme = "https"
			case constants.ETCDMetricsPortHTTP:
				scheme = "http"
			}

			// Check for scheme annotation override
			if schemeOverride, ok := service.Annotations["instana.io/etcd-scheme"]; ok {
				scheme = schemeOverride
			}

			return &port.Port, scheme
		}
	}

	return nil, ""
}

// buildTargetsFromEndpoints builds targets from service endpoint slices
func (r *InstanaAgentReconciler) buildTargetsFromEndpoints(
	ctx context.Context,
	service *corev1.Service,
	metricsPort int32,
	scheme string,
) ([]string, error) {
	// List endpoint slices belonging to the service. EndpointSlice names include a random
	// suffix, so we have to rely on the service label instead of the service name.
	endpointSlices := &discoveryv1.EndpointSliceList{}
	if err := r.client.List(
		ctx,
		endpointSlices,
		client.InNamespace(kubeSystemNamespace),
		client.MatchingLabels(map[string]string{discoveryv1.LabelServiceName: service.Name}),
	); err != nil {
		return nil, err
	}

	targets := make([]string, 0)

	for i := range endpointSlices.Items {
		sliceTargets := buildTargetsFromEndpointSlice(&endpointSlices.Items[i], metricsPort, scheme)
		targets = append(targets, sliceTargets...)
	}

	if len(targets) == 0 {
		legacyTargets, err := r.buildTargetsFromLegacyEndpoints(ctx, service, metricsPort, scheme)
		if err != nil {
			return nil, err
		}
		targets = append(targets, legacyTargets...)
	}

	// Sort targets for consistent comparison with current state
	sort.Strings(targets)

	return targets, nil
}

func buildTargetsFromEndpointSlice(
	endpointSlice *discoveryv1.EndpointSlice,
	metricsPort int32,
	scheme string,
) []string {
	targets := make([]string, 0)

	// Find the metrics port in the endpoint slice
	endpointPort := metricsPort
	for _, port := range endpointSlice.Ports {
		if port.Name != nil && *port.Name == "metrics" && port.Port != nil {
			endpointPort = *port.Port
			break
		}
	}

	// Add targets for each ready endpoint
	for _, endpoint := range endpointSlice.Endpoints {
		// Skip if Ready is explicitly set to false
		// We don't need to check if endpoint.Conditions is nil (it can't be nil since it's a struct value),
		// but we should handle the case where endpoint.Conditions.Ready is nil, which according to the
		// documentation should be interpreted as "true"
		if endpoint.Conditions.Ready != nil && !*endpoint.Conditions.Ready {
			continue
		}
		for _, address := range endpoint.Addresses {
			target := fmt.Sprintf("%s://%s:%d/metrics", scheme, address, endpointPort)
			targets = append(targets, target)
		}
	}

	return targets
}

func (r *InstanaAgentReconciler) buildTargetsFromLegacyEndpoints(
	ctx context.Context,
	service *corev1.Service,
	metricsPort int32,
	scheme string,
) ([]string, error) {
	endpoints := &corev1.Endpoints{}
	if err := r.client.Get(ctx, types.NamespacedName{
		Namespace: kubeSystemNamespace,
		Name:      service.Name,
	}, endpoints); err != nil {
		if apierrors.IsNotFound(err) {
			return []string{}, nil
		}
		return nil, err
	}

	targets := make([]string, 0)

	for _, subset := range endpoints.Subsets {
		endpointPort := metricsPort
		for _, port := range subset.Ports {
			if port.Name == "metrics" {
				endpointPort = port.Port
				break
			}
		}

		for _, address := range subset.Addresses {
			target := fmt.Sprintf("%s://%s:%d/metrics", scheme, address.IP, endpointPort)
			targets = append(targets, target)
		}
	}

	return targets, nil
}

// checkCASecretExists checks if the etcd-ca secret exists in the agent namespace
func (r *InstanaAgentReconciler) checkCASecretExists(
	ctx context.Context,
	agent *instanav1.InstanaAgent,
) bool {
	caSecret := &corev1.Secret{} // pragma: whitelist secret
	err := r.client.Get(ctx, types.NamespacedName{
		Namespace: agent.Namespace,
		Name:      constants.ETCDCASecretName,
	}, caSecret)

	if err == nil {
		r.loggerFor(ctx, agent).Info("Found etcd-ca secret in agent namespace")
		return true
	}

	return false
}
