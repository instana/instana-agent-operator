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
	"fmt"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	backends "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/backends"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/operator_utils"
)

type IstioMeshConfig struct {
	OutboundTrafficPolicy struct {
		Mode string `yaml:"mode"`
	} `yaml:"outboundTrafficPolicy"`
}

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

func (r *InstanaAgentReconciler) getK8SensorBackends(agent *instanav1.InstanaAgent) []backends.K8SensorBackend {
	k8SensorBackends := make([]backends.K8SensorBackend, 0, len(agent.Spec.Agent.AdditionalBackends)+1)
	k8SensorBackends = append(
		k8SensorBackends,
		*backends.NewK8SensorBackend("", agent.Spec.Agent.Key, agent.Spec.Agent.DownloadKey, agent.Spec.Agent.EndpointHost, agent.Spec.Agent.EndpointPort),
	)

	if len(agent.Spec.Agent.AdditionalBackends) == 0 {
		return k8SensorBackends
	}

	for i, additionalBackend := range agent.Spec.Agent.AdditionalBackends {
		k8SensorBackends = append(
			k8SensorBackends,
			*backends.NewK8SensorBackend("-"+strconv.Itoa(i+1), additionalBackend.Key, "", additionalBackend.EndpointHost, additionalBackend.EndpointPort),
		)
	}
	return k8SensorBackends
}

func (r *InstanaAgentReconciler) getIstioOutboundConfigAndNodeIps(ctx context.Context, namespace string, configmap string) (
	bool,
	[]string,
	reconcileReturn,
) {
	log := logf.FromContext(ctx)
	var nodeIPs []string

	log.Info("Check if REGISTRY_ONLY is enabled")
	isIstioRegistryOnlyEnabled := r.checkRegistryOnlyMode(ctx, namespace, configmap)

	if isIstioRegistryOnlyEnabled {
		nodes, err := r.client.ListNodes(ctx)
		if err != nil {
			log.Error(err, "could not list nodes for generating ServiceEntries")
		}
		nodeIPs = getNodeIPs(nodes)
	}

	return isIstioRegistryOnlyEnabled, nodeIPs, reconcileContinue()
}

func (r *InstanaAgentReconciler) checkRegistryOnlyMode(ctx context.Context, namespace string, configmap string) bool {
	istioConfigMap := &corev1.ConfigMap{}
	log := logf.FromContext(ctx)
	log.Info(fmt.Sprintf("Checking Istio ConfigMap %s in namespace %s for outbound traffic policy", configmap, namespace))
	err := r.client.Get(ctx, types.NamespacedName{Name: configmap, Namespace: namespace}, istioConfigMap)
	if err != nil {
		log.Error(err, "Failed fetching istio ConfigMap")
		return false
	}
	if istioConfigMap.Data == nil {
		log.Info(fmt.Sprintf("Istio configmap %s in namespace %s data in nil", configmap, namespace))
		return false
	}
	meshConfigData, ok := istioConfigMap.Data["mesh"]
	if !ok {
		return false
	}

	var meshConfig IstioMeshConfig
	log.Info("Unmarshalling config data")
	err = yaml.Unmarshal([]byte(meshConfigData), &meshConfig)
	if err != nil {
		log.Error(err, "Unmarshalling config data ERROR")
		return false
	}
	log.Info("Checking if policy is REGISTRY_ONLY")

	return strings.EqualFold(meshConfig.OutboundTrafficPolicy.Mode, "REGISTRY_ONLY")
}

func getNodeIPs(nodes *corev1.NodeList) []string {
	var nodeIPs []string
	for _, node := range nodes.Items {
		for _, address := range node.Status.Addresses {
			if address.Type == corev1.NodeInternalIP {
				nodeIPs = append(nodeIPs, address.Address)
			}
		}
	}
	return nodeIPs
}

func (r *InstanaAgentReconciler) loggerFor(ctx context.Context, agent *instanav1.InstanaAgent) logr.Logger {
	return logf.FromContext(ctx).WithValues(
		"Generation",
		agent.Generation,
		"UID",
		agent.UID,
	)
}
