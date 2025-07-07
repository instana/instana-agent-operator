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

package v1

import (
	"fmt"
	"reflect"

	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:openapi-gen=true

// InstanaAgentRemoteSpec defines the desired state of the Instana Agent Remote
type InstanaAgentRemoteSpec struct {
	// Agent deployment specific fields.
	// +kubebuilder:validation:Optional
	Agent BaseAgentSpec `json:"agent"`

	Cluster Name `json:"cluster,omitempty"` //inherit from main agent

	// Name of the zone in which the host(s) will be displayed on the map. Optional, but then 'cluster.name' must be specified.
	Zone Name `json:"zone,omitempty"`

	// Specifies whether RBAC resources should be created.
	// +kubebuilder:validation:Optional
	Rbac Create `json:"rbac,omitempty"`

	// Specifies whether a ServiceAccount should be created (default `true`).
	// +kubebuilder:validation:Optional
	ServiceAccountSpec `json:"serviceAccount,omitempty"`

	// +kubebuilder:validation:Optional
	ResourceRequirements `json:",omitempty"`

	// Supply Agent configuration e.g. for configuring certain Sensors.
	// +kubebuilder:validation:Required
	ConfigurationYaml string `json:"remote_configuration_yaml,omitempty"`

	// Supply Agent configuration values instead of inheriting from host.
	// +kubebuilder:validation:Optional,
	ManualSetup *bool `json:"manual_setup,omitempty"`

	Hostname string `json:"hostname,omitempty"`
}

// +k8s:openapi-gen=true

// InstanaAgentRemoteOperatorState type representing the running state of the Agent Operator itself.
type InstanaAgentRemoteOperatorState string

const (
	RemoteOperatorStateRunning  InstanaAgentRemoteOperatorState = "Running"
	RemoteOperatorStateUpdating InstanaAgentRemoteOperatorState = "Updating"
	RemoteOperatorStateFailed   InstanaAgentRemoteOperatorState = "Failed"
)

// +k8s:openapi-gen=true

type InstanaAgentRemoteStatus struct {
	ConfigSecret       ResourceInfo       `json:"configsecret,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
	ObservedGeneration *int64             `json:"observedGeneration,omitempty"`
	OperatorVersion    *SemanticVersion   `json:"operatorVersion,omitempty"`
	Deployment         ResourceInfo       `json:"deployment,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=agentsremote,singular=agentremote,shortName=ar,scope=Namespaced,categories=monitoring;
// +kubebuilder:storageversion
// +operator-sdk:csv:customresourcedefinitions:displayName="Remote Instana Agent",resources={{Deployment,apps/v1,InstanaAgentRemote},{Pod,v1,InstanaAgentRemote},{Secret,v1,InstanaAgentRemote}}

type InstanaAgentRemote struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstanaAgentRemoteSpec   `json:"spec,omitempty"`
	Status InstanaAgentRemoteStatus `json:"status,omitempty"`
}

func (in *InstanaAgentRemote) InheritDefault(agent InstanaAgent) {
	desiredEndpointHost := optional.Of(agent.Spec.Agent.EndpointHost).GetOrDefault("ingress-red-saas.instana.io")
	inheritString(&in.Spec.Agent.EndpointHost, &desiredEndpointHost)

	desiredEndpointPort := optional.Of(agent.Spec.Agent.EndpointPort).GetOrDefault("443")
	inheritString(&in.Spec.Agent.EndpointPort, &desiredEndpointPort)

	desiredImageName := optional.Of(agent.Spec.Agent.ImageSpec.Name).GetOrDefault("icr.io/instana/agent")
	inheritString(&in.Spec.Agent.ImageSpec.Name, &desiredImageName)

	desiredImageTag := optional.Of(agent.Spec.Agent.ImageSpec.Tag).GetOrDefault("latest")
	inheritString(&in.Spec.Agent.ImageSpec.Tag, &desiredImageTag)

	desiredPullPolicy := optional.Of(agent.Spec.Agent.ImageSpec.PullPolicy).GetOrDefault(corev1.PullAlways)
	inheritPullPolicy(&in.Spec.Agent.ImageSpec.PullPolicy, &desiredPullPolicy)

	desiredRbac := optional.Of(agent.Spec.Rbac.Create).GetOrDefault(pointer.To(true))
	inheritBoolPointer(&in.Spec.Rbac.Create, &desiredRbac)

	desiredSA := optional.Of(agent.Spec.ServiceAccountSpec.Create.Create).GetOrDefault(pointer.To(true))
	inheritBoolPointer(&in.Spec.ServiceAccountSpec.Create.Create, &desiredSA)

	optional.ValueOrDefault(&in.Spec.Hostname, fmt.Sprintf("instana-agent-remote-%s-%s", in.Namespace, in.Name))

	inheritString(&in.Spec.Agent.ConfigurationYaml, &in.Spec.ConfigurationYaml)
	inheritString(&in.Spec.Cluster.Name, &agent.Spec.Cluster.Name)
	inheritString(&in.Spec.Agent.Key, &agent.Spec.Agent.Key)
	inheritString(&in.Spec.Agent.DownloadKey, &agent.Spec.Agent.DownloadKey)
	inheritString(&in.Spec.Agent.KeysSecret, &agent.Spec.Agent.KeysSecret)
	inheritString(&in.Spec.Agent.ListenAddress, &agent.Spec.Agent.ListenAddress)
	inheritInt(&in.Spec.Agent.MinReadySeconds, &agent.Spec.Agent.MinReadySeconds)
	inheritString(&in.Spec.Agent.ProxyHost, &agent.Spec.Agent.ProxyHost)
	inheritString(&in.Spec.Agent.ProxyPassword, &agent.Spec.Agent.ProxyPassword)
	inheritString(&in.Spec.Agent.ProxyPort, &agent.Spec.Agent.ProxyPort)
	inheritString(&in.Spec.Agent.ProxyProtocol, &agent.Spec.Agent.ProxyProtocol)
	inheritBool(&in.Spec.Agent.ProxyUseDNS, &agent.Spec.Agent.ProxyUseDNS)
	inheritString(&in.Spec.Agent.ProxyUser, &agent.Spec.Agent.ProxyUser)
	inheritString(&in.Spec.Agent.RedactKubernetesSecrets, &agent.Spec.Agent.RedactKubernetesSecrets)
	inheritString(&in.Spec.Agent.MvnRepoFeaturesPath, &agent.Spec.Agent.MvnRepoFeaturesPath)
	inheritString(&in.Spec.Agent.MvnRepoSharedPath, &agent.Spec.Agent.MvnRepoSharedPath)
	inheritString(&in.Spec.Agent.MvnRepoUrl, &agent.Spec.Agent.MvnRepoUrl)
	inheritString(&in.Spec.Agent.MirrorReleaseRepoPassword, &agent.Spec.Agent.MirrorReleaseRepoPassword)
	inheritString(&in.Spec.Agent.MirrorReleaseRepoUrl, &agent.Spec.Agent.MirrorReleaseRepoUrl)
	inheritString(&in.Spec.Agent.MirrorReleaseRepoUsername, &agent.Spec.Agent.MirrorReleaseRepoUsername)
	inheritString(&in.Spec.Agent.MirrorSharedRepoPassword, &agent.Spec.Agent.MirrorSharedRepoPassword)
	inheritString(&in.Spec.Agent.MirrorSharedRepoUrl, &agent.Spec.Agent.MirrorSharedRepoUrl)
	inheritString(&in.Spec.Agent.MirrorSharedRepoUsername, &agent.Spec.Agent.MirrorSharedRepoUsername)

	if !reflect.DeepEqual(in.Spec.Agent.AdditionalBackends, agent.Spec.Agent.AdditionalBackends) {
		in.Spec.Agent.AdditionalBackends = agent.Spec.Agent.AdditionalBackends
	}

	if !reflect.DeepEqual(in.Spec.Agent.TlsSpec, agent.Spec.Agent.TlsSpec) {
		in.Spec.Agent.TlsSpec = agent.Spec.Agent.TlsSpec
	}

	if !reflect.DeepEqual(in.Spec.Agent.Pod.ResourceRequirements, in.Spec.ResourceRequirements) {
		in.Spec.Agent.Pod.ResourceRequirements = in.Spec.ResourceRequirements
	}
}

func (in *InstanaAgentRemote) Default() {
	optional.ValueOrDefault(&in.Spec.Agent.ConfigurationYaml, in.Spec.ConfigurationYaml)
	optional.ValueOrDefault(&in.Spec.Agent.EndpointHost, "ingress-red-saas.instana.io")
	optional.ValueOrDefault(&in.Spec.Agent.EndpointPort, "443")
	optional.ValueOrDefault(&in.Spec.Agent.ImageSpec.Name, "icr.io/instana/agent")
	optional.ValueOrDefault(&in.Spec.Agent.ImageSpec.Tag, "latest")
	optional.ValueOrDefault(&in.Spec.Agent.ImageSpec.PullPolicy, corev1.PullAlways)
	optional.ValueOrDefault(&in.Spec.Rbac.Create, pointer.To(true))
	optional.ValueOrDefault(&in.Spec.ServiceAccountSpec.Create.Create, pointer.To(true))
	optional.ValueOrDefault(&in.Spec.Agent.Pod.ResourceRequirements, in.Spec.ResourceRequirements)
	optional.ValueOrDefault(&in.Spec.Hostname, fmt.Sprintf("instana-agent-remote-%s-%s", in.Namespace, in.Name))
}

// +kubebuilder:object:root=true

type InstanaAgentRemoteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InstanaAgentRemote `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InstanaAgentRemote{}, &InstanaAgentRemoteList{})
}

func inheritString(target *string, source *string) {
	if source != nil && *source != "" && (target == nil || *target != *source) {
		*target = *source
	}
}

func inheritBool(target *bool, source *bool) {
	if source != nil && (target == nil || *target != *source) {
		*target = *source
	}
}

func inheritBoolPointer(target **bool, source **bool) {
	if source != nil && *source != nil && (target == nil || *target == nil || **target != **source) {
		*target = *source
	}
}

func inheritInt(target *int, source *int) {
	if source != nil && *source != 0 && (target == nil || *target != *source) {
		*target = *source
	}
}

func inheritPullPolicy(target *corev1.PullPolicy, source *corev1.PullPolicy) {
	if source != nil && *source != "" && (target == nil || *target != *source) {
		*target = *source
	}
}
