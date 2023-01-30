/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package v1beta1

import (
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"

	v1 "github.com/instana/instana-agent-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

const (
	DefaultLeaderKey = "com.instana.plugin.kubernetes.leader"
)

// ConvertTo converts this InstanaAgent to the Hub version (v1).
func (src *InstanaAgent) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1.InstanaAgent)

	src.convertInternalSpecTo(dst)
	dst.ObjectMeta = src.ObjectMeta
	src.convertStatusTo(&dst.Status)

	return nil
}

func (src *InstanaAgent) convertInternalSpecTo(dst *v1.InstanaAgent) {
	srcSpec := src.Spec

	// Helm charts don't support multiple configuration files so merge them assuming it's all the configuration.yaml
	if len(srcSpec.ConfigurationFiles) > 0 {
		var sb strings.Builder
		for _, configFile := range srcSpec.ConfigurationFiles {
			sb.WriteString(configFile)
			sb.WriteString("\n")
		}
		dst.Spec.Agent.ConfigurationYaml = sb.String()
	}

	dst.Spec.Zone.Name = srcSpec.AgentZoneName
	dst.Spec.Cluster.Name = srcSpec.ClusterName
	dst.Spec.Agent.Key = srcSpec.AgentKey
	dst.Spec.Agent.DownloadKey = srcSpec.AgentDownloadKey
	dst.Spec.Agent.EndpointHost = srcSpec.AgentEndpointHost
	dst.Spec.Agent.EndpointPort = strconv.FormatUint(uint64(srcSpec.AgentEndpointPort), 10)
	dst.Spec.Agent.Host.Repository = srcSpec.AgentRepository

	// Cannot specify names for e.g. ClusterRole and ServiceAccount in Helm, so omit those
	//AgentClusterRoleName        string            `json:"agent.clusterRoleName,omitempty"`
	//AgentClusterRoleBindingName string            `json:"agent.clusterRoleBindingName,omitempty"`
	//AgentServiceAccountName     string            `json:"agent.serviceAccountName,omitempty"`
	//AgentSecretName             string            `json:"agent.secretName,omitempty"`
	//AgentDaemonSetName          string            `json:"agent.daemonSetName,omitempty"`
	//AgentConfigMapName          string            `json:"agent.configMapName,omitempty"`

	if len(srcSpec.AgentImageName) > 0 {
		lastIndex := strings.LastIndex(srcSpec.AgentImageName, ":")
		if lastIndex > 0 {
			dst.Spec.Agent.ImageSpec.Name = srcSpec.AgentImageName[:lastIndex]
			dst.Spec.Agent.ImageSpec.Tag = srcSpec.AgentImageName[lastIndex+1:]
		} else {
			// No separator for Tag found, assume only name
			dst.Spec.Agent.ImageSpec.Name = srcSpec.AgentImageName
		}
	}

	dst.Spec.Agent.ImageSpec.PullPolicy = srcSpec.AgentImagePullPolicy

	// Build up the object, so we can directly store Quantity values as long as they're not 0
	dst.Spec.Agent.Pod.ResourceRequirements = corev1.ResourceRequirements{
		Requests: make(map[corev1.ResourceName]resource.Quantity),
		Limits:   make(map[corev1.ResourceName]resource.Quantity),
	}
	if !srcSpec.AgentCpuReq.IsZero() {
		dst.Spec.Agent.Pod.ResourceRequirements.Requests[corev1.ResourceCPU] = srcSpec.AgentCpuReq
	}
	if !srcSpec.AgentMemReq.IsZero() {
		dst.Spec.Agent.Pod.ResourceRequirements.Requests[corev1.ResourceMemory] = srcSpec.AgentMemReq
	}
	if !srcSpec.AgentCpuLim.IsZero() {
		dst.Spec.Agent.Pod.ResourceRequirements.Limits[corev1.ResourceCPU] = srcSpec.AgentCpuLim
	}
	if !srcSpec.AgentMemLim.IsZero() {
		dst.Spec.Agent.Pod.ResourceRequirements.Limits[corev1.ResourceMemory] = srcSpec.AgentMemLim
	}

	dst.Spec.Rbac.Create = srcSpec.AgentRbacCreate
	dst.Spec.OpenTelemetry.Enabled.Enabled = srcSpec.OpenTelemetryEnabled

	dst.Spec.Agent.TlsSpec.SecretName = srcSpec.AgentTlsSecretName
	dst.Spec.Agent.TlsSpec.Certificate = srcSpec.AgentTlsCertificate
	dst.Spec.Agent.TlsSpec.Key = srcSpec.AgentTlsKey

	dst.Spec.Agent.Env = srcSpec.AgentEnv
}

func (src *InstanaAgent) convertStatusTo(dstStatus *v1.InstanaAgentStatus) {
	dstStatus.ConfigMap = convertResourceInfoTo(src.Status.ConfigMap)
	dstStatus.DaemonSet = convertResourceInfoTo(src.Status.DaemonSet)
	dstStatus.ServiceAccount = convertResourceInfoTo(src.Status.ServiceAccount)
	dstStatus.ClusterRole = convertResourceInfoTo(src.Status.ClusterRoleBinding)
	dstStatus.ClusterRoleBinding = convertResourceInfoTo(src.Status.ClusterRoleBinding)
	dstStatus.Secret = convertResourceInfoTo(src.Status.Secret)
	if dstStatus.LeadingAgentPod == nil {
		dstStatus.LeadingAgentPod = make(map[string]v1.ResourceInfo, 1)
	}
	dstStatus.LeadingAgentPod[DefaultLeaderKey] = convertResourceInfoTo(src.Status.LeadingAgentPod)
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
	if len(src.Spec.Agent.EndpointPort) > 0 {
		if u, err := strconv.ParseUint(src.Spec.Agent.EndpointPort, 10, 16); err == nil {
			dst.Spec.AgentEndpointPort = uint16(u)
		} else {
			return err
		}
	}
	dst.Spec.AgentRepository = src.Spec.Agent.Host.Repository

	// Cannot specify names for e.g. ClusterRole and ServiceAccount in Helm, so omit those
	//AgentClusterRoleName        string            `json:"agent.clusterRoleName,omitempty"`
	//AgentClusterRoleBindingName string            `json:"agent.clusterRoleBindingName,omitempty"`
	//AgentServiceAccountName     string            `json:"agent.serviceAccountName,omitempty"`
	//AgentSecretName             string            `json:"agent.secretName,omitempty"`
	//AgentDaemonSetName          string            `json:"agent.daemonSetName,omitempty"`
	//AgentConfigMapName          string            `json:"agent.configMapName,omitempty"`

	if len(src.Spec.Agent.ImageSpec.Tag) > 0 {
		dst.Spec.AgentImageName = src.Spec.Agent.ImageSpec.Name + ":" + src.Spec.Agent.ImageSpec.Tag
	} else {
		dst.Spec.AgentImageName = src.Spec.Agent.ImageSpec.Name
	}

	dst.Spec.AgentImagePullPolicy = src.Spec.Agent.ImageSpec.PullPolicy

	if value, ok := src.Spec.Agent.Pod.ResourceRequirements.Requests[corev1.ResourceCPU]; ok {
		dst.Spec.AgentCpuReq = value
	}
	if value, ok := src.Spec.Agent.Pod.ResourceRequirements.Requests[corev1.ResourceMemory]; ok {
		dst.Spec.AgentMemReq = value
	}
	if value, ok := src.Spec.Agent.Pod.ResourceRequirements.Limits[corev1.ResourceCPU]; ok {
		dst.Spec.AgentCpuLim = value
	}
	if value, ok := src.Spec.Agent.Pod.ResourceRequirements.Limits[corev1.ResourceMemory]; ok {
		dst.Spec.AgentMemLim = value
	}

	dst.Spec.AgentRbacCreate = src.Spec.Rbac.Create
	dst.Spec.OpenTelemetryEnabled = src.Spec.OpenTelemetry.Enabled.Enabled

	dst.Spec.AgentTlsSecretName = src.Spec.Agent.TlsSpec.SecretName
	dst.Spec.AgentTlsCertificate = src.Spec.Agent.TlsSpec.Certificate
	dst.Spec.AgentTlsKey = src.Spec.Agent.TlsSpec.Key

	dst.Spec.AgentEnv = src.Spec.Agent.Env

	dst.ObjectMeta = src.ObjectMeta
	dst.convertStatusFrom(src.Status)

	return nil
}

func (dst *InstanaAgent) convertStatusFrom(srcStatus v1.InstanaAgentStatus) {
	dst.Status.ConfigMap = convertResourceInfoFrom(srcStatus.ConfigMap)
	dst.Status.DaemonSet = convertResourceInfoFrom(srcStatus.DaemonSet)
	dst.Status.ServiceAccount = convertResourceInfoFrom(srcStatus.ServiceAccount)
	dst.Status.ClusterRole = convertResourceInfoFrom(srcStatus.ClusterRoleBinding)
	dst.Status.ClusterRoleBinding = convertResourceInfoFrom(srcStatus.ClusterRoleBinding)
	dst.Status.Secret = convertResourceInfoFrom(srcStatus.Secret)
	if leader, ok := srcStatus.LeadingAgentPod[DefaultLeaderKey]; ok {
		dst.Status.LeadingAgentPod = convertResourceInfoFrom(leader)
	} else {
		// If the expected key is not there, pick just the first value we find
		for _, value := range srcStatus.LeadingAgentPod {
			dst.Status.LeadingAgentPod = convertResourceInfoFrom(value)
		}
	}
}

func convertResourceInfoFrom(src v1.ResourceInfo) ResourceInfo {
	dstInfo := &ResourceInfo{
		Name: src.Name,
		UID:  src.UID,
	}
	return *dstInfo
}
