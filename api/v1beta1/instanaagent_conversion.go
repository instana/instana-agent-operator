/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package v1beta1

import (
	"strconv"
	"strings"

	v1 "github.com/instana/instana-agent-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertTo converts this InstanaAgent to the Hub version (v1).
func (src *InstanaAgent) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1.InstanaAgent)

	// Helm charts don't support multiple configuration files so merge them assuming it's all the configuration.yaml
	var sb strings.Builder
	for _, configFile := range src.Spec.ConfigurationFiles {
		sb.WriteString(configFile)
		sb.WriteString("\n")
	}
	dst.Spec.Agent.ConfigurationYaml = sb.String()

	dst.Spec.Zone.Name = src.Spec.AgentZoneName
	dst.Spec.Cluster.Name = src.Spec.ClusterName
	dst.Spec.Agent.Key = src.Spec.AgentKey
	dst.Spec.Agent.DownloadKey = src.Spec.AgentDownloadKey
	dst.Spec.Agent.EndpointHost = src.Spec.AgentEndpointHost
	dst.Spec.Agent.EndpointPort = strconv.FormatUint(uint64(src.Spec.AgentEndpointPort), 10)
	dst.Spec.Agent.Host.Repository = src.Spec.AgentRepository

	// Cannot specify names for e.g. ClusterRole and ServiceAccount in Helm, so omit those
	//AgentClusterRoleName        string            `json:"agent.clusterRoleName,omitempty"`
	//AgentClusterRoleBindingName string            `json:"agent.clusterRoleBindingName,omitempty"`
	//AgentServiceAccountName     string            `json:"agent.serviceAccountName,omitempty"`
	//AgentSecretName             string            `json:"agent.secretName,omitempty"`
	//AgentDaemonSetName          string            `json:"agent.daemonSetName,omitempty"`
	//AgentConfigMapName          string            `json:"agent.configMapName,omitempty"`

	lastIndex := strings.LastIndex(src.Spec.AgentImageName, ":")
	if lastIndex > 0 {
		dst.Spec.Agent.Image.Name = src.Spec.AgentImageName[:lastIndex]
		dst.Spec.Agent.Image.Tag = src.Spec.AgentImageName[lastIndex+1:]
	} else {
		// No separator for Tag found, assume only name
		dst.Spec.Agent.Image.Name = src.Spec.AgentImageName
	}

	dst.Spec.Agent.Image.PullPolicy = src.Spec.AgentImagePullPolicy

	dst.Spec.Agent.Pod.ResourceRequirements.Requests[corev1.ResourceCPU] = src.Spec.AgentCpuReq
	dst.Spec.Agent.Pod.ResourceRequirements.Requests[corev1.ResourceMemory] = src.Spec.AgentMemReq
	dst.Spec.Agent.Pod.ResourceRequirements.Limits[corev1.ResourceCPU] = src.Spec.AgentCpuLim
	dst.Spec.Agent.Pod.ResourceRequirements.Limits[corev1.ResourceMemory] = src.Spec.AgentMemLim

	dst.Spec.Rbac.Create = src.Spec.AgentRbacCreate
	dst.Spec.OpenTelemetry.Enabled = src.Spec.OpenTelemetryEnabled

	dst.Spec.Agent.Env = src.Spec.AgentEnv

	dst.ObjectMeta = src.ObjectMeta

	err := src.convertStatusTo(dst.Status)
	return err
}

func (src *InstanaAgent) convertStatusTo(dstStatus v1.InstanaAgentStatus) error {
	dstStatus.ConfigMap = convertResourceInfoTo(src.Status.ConfigMap)
	dstStatus.DaemonSet = convertResourceInfoTo(src.Status.DaemonSet)
	dstStatus.LeadingAgentPod = convertResourceInfoTo(src.Status.LeadingAgentPod)
	return nil
}

func convertResourceInfoTo(src ResourceInfo) v1.ResourceInfo {
	dstInfo := &v1.ResourceInfo{
		Name: src.Name,
		UID:  src.UID,
	}
	return *dstInfo
}

// ConvertFrom converts from the Hub version (v1) to this version.
func (dst *InstanaAgent) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1.InstanaAgent)

	dst.Spec.ConfigurationFiles = make(map[string]string)
	if len(src.Spec.Agent.ConfigurationYaml) > 0 {
		dst.Spec.ConfigurationFiles["configuration.yaml"] = src.Spec.Agent.ConfigurationYaml
	}

	dst.Spec.AgentZoneName = src.Spec.Zone.Name
	dst.Spec.ClusterName = src.Spec.Cluster.Name
	dst.Spec.AgentKey = src.Spec.Agent.Key
	dst.Spec.AgentDownloadKey = src.Spec.Agent.DownloadKey
	dst.Spec.AgentEndpointHost = src.Spec.Agent.EndpointHost
	if u, err := strconv.ParseUint(src.Spec.Agent.EndpointPort, 10, 16); err == nil {
		dst.Spec.AgentEndpointPort = uint16(u)
	} else {
		return err
	}
	dst.Spec.AgentRepository = src.Spec.Agent.Host.Repository

	// Cannot specify names for e.g. ClusterRole and ServiceAccount in Helm, so omit those
	//AgentClusterRoleName        string            `json:"agent.clusterRoleName,omitempty"`
	//AgentClusterRoleBindingName string            `json:"agent.clusterRoleBindingName,omitempty"`
	//AgentServiceAccountName     string            `json:"agent.serviceAccountName,omitempty"`
	//AgentSecretName             string            `json:"agent.secretName,omitempty"`
	//AgentDaemonSetName          string            `json:"agent.daemonSetName,omitempty"`
	//AgentConfigMapName          string            `json:"agent.configMapName,omitempty"`

	if len(src.Spec.Agent.Image.Tag) > 0 {
		dst.Spec.AgentImageName = src.Spec.Agent.Image.Name + ":" + src.Spec.Agent.Image.Tag
	} else {
		dst.Spec.AgentImageName = src.Spec.Agent.Image.Name
	}

	dst.Spec.AgentImagePullPolicy = src.Spec.Agent.Image.PullPolicy

	if _, ok := src.Spec.Agent.Pod.ResourceRequirements.Requests[corev1.ResourceCPU]; ok {
		dst.Spec.AgentCpuReq = src.Spec.Agent.Pod.ResourceRequirements.Requests[corev1.ResourceCPU]
	}
	if _, ok := src.Spec.Agent.Pod.ResourceRequirements.Requests[corev1.ResourceMemory]; ok {
		dst.Spec.AgentMemReq = src.Spec.Agent.Pod.ResourceRequirements.Requests[corev1.ResourceMemory]
	}
	if _, ok := src.Spec.Agent.Pod.ResourceRequirements.Limits[corev1.ResourceCPU]; ok {
		dst.Spec.AgentCpuLim = src.Spec.Agent.Pod.ResourceRequirements.Limits[corev1.ResourceCPU]
	}
	if _, ok := src.Spec.Agent.Pod.ResourceRequirements.Limits[corev1.ResourceMemory]; ok {
		dst.Spec.AgentMemLim = src.Spec.Agent.Pod.ResourceRequirements.Limits[corev1.ResourceMemory]
	}

	dst.Spec.AgentRbacCreate = src.Spec.Rbac.Create
	dst.Spec.OpenTelemetryEnabled = src.Spec.OpenTelemetry.Enabled

	dst.Spec.AgentEnv = src.Spec.Agent.Env

	dst.ObjectMeta = src.ObjectMeta

	err := dst.convertStatusFrom(src.Status)
	return err
}

func (dst *InstanaAgent) convertStatusFrom(srcStatus v1.InstanaAgentStatus) error {
	dst.Status.ConfigMap = convertResourceInfoFrom(srcStatus.ConfigMap)
	dst.Status.DaemonSet = convertResourceInfoFrom(srcStatus.DaemonSet)
	dst.Status.LeadingAgentPod = convertResourceInfoFrom(srcStatus.LeadingAgentPod)
	return nil
}

func convertResourceInfoFrom(src v1.ResourceInfo) ResourceInfo {
	dstInfo := &ResourceInfo{
		Name: src.Name,
		UID:  src.UID,
	}
	return *dstInfo
}
