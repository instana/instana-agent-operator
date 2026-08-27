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

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
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
	shouldSkip, err := r.etcdDiscoverer.ShouldSkipDiscovery(ctx, agent)
	if err != nil {
		return nil, err
	}
	if shouldSkip {
		log.Info("Skipping ETCD discovery based on configuration or environment")
		return nil, nil
	}

	// Step 2: Find etcd service
	etcdService, err := r.etcdDiscoverer.FindETCDService(ctx, log)
	if err != nil {
		return nil, err
	}
	if etcdService == nil {
		log.Info("No ETCD service found in kube-system namespace")
		return nil, nil
	}

	log.Info("Found etcd service", "name", etcdService.Name)

	// Step 3: Find metrics port and determine scheme
	metricsPortPtr, scheme := r.etcdDiscoverer.FindMetricsPortAndScheme(etcdService)
	if metricsPortPtr == nil {
		log.Info("No metrics port found in etcd service")
		return nil, nil
	}
	metricsPort := *metricsPortPtr

	// Step 4: Get endpoints and build targets
	targets, err := r.etcdDiscoverer.BuildTargetsFromEndpoints(
		ctx,
		etcdService,
		metricsPort,
		scheme,
	)
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		log.Info("No endpoints found for etcd service")
		return nil, nil
	}

	// Step 5: Check for CA secret and return results
	caSecretExists := r.etcdDiscoverer.CheckCASecretExists(ctx, agent)

	log.Info("Discovered etcd targets", "targets", targets, "caFound", caSecretExists)

	return &DiscoveredETCDTargets{
		Targets: targets,
		CAFound: caSecretExists,
	}, nil
}
