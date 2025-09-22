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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/operator_utils"
)

// DiscoveredETCDTargets holds information about discovered ETCD endpoints
type DiscoveredETCDTargets struct {
	Targets []string
	CAFound bool
}

// DiscoverETCDEndpoints attempts to discover ETCD endpoints in vanilla Kubernetes clusters.
// Discovery logic:
// 1. Skip if on OpenShift or if targets are already specified in the CR
// 2. Look for a service in kube-system with label component=etcd
// 3. Fallback to a service with "etcd" in the name
// 4. Find a port named "metrics"
// 5. Determine scheme based on port number (2379 -> https, 2381 -> http)
// 6. Override scheme if service has annotation instana.io/etcd-scheme
// 7. Get endpoints for the service
// 8. Build targets as scheme://ip:port/metrics
// 9. Check if CA secret "etcd-ca" exists in agent namespace
func (r *InstanaAgentReconciler) DiscoverETCDEndpoints(ctx context.Context, agent *instanav1.InstanaAgent) (*DiscoveredETCDTargets, error) {
	log := r.loggerFor(ctx, agent)

	// Skip discovery if we're on OpenShift or if targets are already specified in the CR
	operatorUtils := operator_utils.NewOperatorUtils(ctx, r.client, agent, nil)
	isOpenShift, isOpenShiftRes := r.isOpenShift(ctx, operatorUtils)
	if isOpenShiftRes.suppliesReconcileResult() {
		log.Error(nil, "Failed to determine if cluster is OpenShift")
		return nil, fmt.Errorf("failed to determine if cluster is OpenShift")
	}

	if isOpenShift {
		log.Info("Skipping ETCD discovery on OpenShift cluster")
		return nil, nil
	}

	if len(agent.Spec.K8sSensor.ETCD.Targets) > 0 {
		log.Info("Using ETCD targets from CR spec", "targets", agent.Spec.K8sSensor.ETCD.Targets)
		return &DiscoveredETCDTargets{
			Targets: agent.Spec.K8sSensor.ETCD.Targets,
		}, nil
	}

	// Try to get the etcd service directly by name
	etcdService := &corev1.Service{}
	err := r.client.Get(ctx, types.NamespacedName{
		Namespace: "kube-system",
		Name:      "etcd",
	}, etcdService)

	if err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "Error getting etcd service")
			return nil, err
		}

		// If not found by name, try etcd-metrics
		err = r.client.Get(ctx, types.NamespacedName{
			Namespace: "kube-system",
			Name:      "etcd-metrics",
		}, etcdService)

		if err != nil {
			if !apierrors.IsNotFound(err) {
				log.Error(err, "Error getting etcd-metrics service")
				return nil, err
			}

			// If still not found, try etcd-k8s
			err = r.client.Get(ctx, types.NamespacedName{
				Namespace: "kube-system",
				Name:      "etcd-k8s",
			}, etcdService)

			if err != nil {
				if !apierrors.IsNotFound(err) {
					log.Error(err, "Error getting etcd-k8s service")
					return nil, err
				}

				log.Info("No etcd service found in kube-system namespace")
				return nil, nil
			}
		}
	}

	log.Info("Found etcd service", "name", etcdService.Name)


	// Find metrics port and determine scheme
	var metricsPort int32
	var scheme string

	for _, port := range etcdService.Spec.Ports {
		if port.Name == "metrics" {
			metricsPort = port.Port

			// Determine scheme based on port number
			if metricsPort == 2379 {
				scheme = "https"
			} else if metricsPort == 2381 {
				scheme = "http"
			} else {
				// Default to https for unknown ports
				scheme = "https"
			}

			// Check for scheme annotation override
			if schemeOverride, ok := etcdService.Annotations["instana.io/etcd-scheme"]; ok {
				scheme = schemeOverride
			}

			break
		}
	}

	if metricsPort == 0 {
		log.Info("No metrics port found in etcd service")
		return nil, nil
	}

	// Get endpoints for the service
	endpoints := &corev1.Endpoints{}
	if err := r.client.Get(ctx, types.NamespacedName{
		Namespace: "kube-system",
		Name:      etcdService.Name,
	}, endpoints); err != nil {
		log.Error(err, "Failed to get endpoints for etcd service")
		return nil, err
	}

	// Build targets from endpoints
	var targets []string

	for _, subset := range endpoints.Subsets {
		// Find the metrics port in the endpoint subset
		var endpointPort int32
		for _, port := range subset.Ports {
			if port.Name == "metrics" {
				endpointPort = port.Port
				break
			}
		}

		// If no metrics port found in endpoints, use the service port
		if endpointPort == 0 {
			endpointPort = metricsPort
		}

		// Add targets for each address
		for _, address := range subset.Addresses {
			target := fmt.Sprintf("%s://%s:%d/metrics", 
				scheme,
				address.IP,
				endpointPort)
			targets = append(targets, target)
		}
	}

	if len(targets) == 0 {
		log.Info("No endpoints found for etcd service")
		return nil, nil
	}

	// Check if CA secret exists in agent namespace
	caSecretExists := false
	caSecret := &corev1.Secret{}
	if err := r.client.Get(ctx, types.NamespacedName{
		Namespace: agent.Namespace,
		Name:      "etcd-ca",
	}, caSecret); err == nil {
		caSecretExists = true
		log.Info("Found etcd-ca secret in agent namespace")
	}

	log.Info("Discovered etcd targets", "targets", targets, "caFound", caSecretExists)

	return &DiscoveredETCDTargets{
		Targets: targets,
		CAFound: caSecretExists,
	}, nil
}
