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
package deployment

import (
	"fmt"
	"regexp"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/hash"
	backends "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/backends"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/env"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/volume"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/transformations"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/status"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

const (
	componentName = constants.ComponentInstanaAgentRemote
)

type deploymentBuilder struct {
	*instanav1.InstanaAgentRemote
	statusManager status.InstanaAgentRemoteStatusManager
	helpers.RemoteHelpers
	transformations.PodSelectorLabelGeneratorRemote
	hash.JsonHasher
	env.EnvBuilderRemote
	volume.VolumeBuilderRemote
	backend    backends.RemoteSensorBackend
	keysSecret *corev1.Secret
	zone       *instanav1.Zone
}

func (d *deploymentBuilder) ComponentName() string {
	return componentName
}

func (d *deploymentBuilder) IsNamespaced() bool {
	return true
}

func (d *deploymentBuilder) getPodTemplateLabels() map[string]string {
	podLabels := optional.Of(d.InstanaAgentRemote.Spec.Agent.Pod.Labels).GetOrDefault(map[string]string{})
	podLabels[constants.LabelAgentMode] = string(optional.Of(d.InstanaAgentRemote.Spec.Agent.Mode).GetOrDefault(instanav1.APM))

	return d.GetPodLabels(podLabels)
}

func (d *deploymentBuilder) getEnvVars() []corev1.EnvVar {
	baseEnvVars := d.EnvBuilderRemote.Build(
		env.AgentModeEnvRemote,
		env.ZoneNameEnvRemote,
		env.AgentEndpointEnvRemote,
		env.AgentEndpointPortEnvRemote,
		env.MavenRepoURLEnvRemote,
		env.MavenRepoFeaturesPathRemote,
		env.MavenRepoSharedPathRemote,
		env.MirrorReleaseRepoUrlEnvRemote,
		env.MirrorReleaseRepoUsernameEnvRemote,
		env.MirrorReleaseRepoPasswordEnvRemote,
		env.MirrorSharedRepoUrlEnvRemote,
		env.MirrorSharedRepoUsernameEnvRemote,
		env.MirrorSharedRepoPasswordEnvRemote,
		env.ProxyHostEnvRemote,
		env.ProxyPortEnvRemote,
		env.ProxyProtocolEnvRemote,
		env.ProxyUserEnvRemote,
		env.ProxyPasswordEnvRemote,
		env.ProxyUseDNSEnvRemote,
		env.ListenAddressEnvRemote,
		env.RedactK8sSecretsEnvRemote,
		env.ConfigPathEnvRemote,
		env.EntrypointSkipBackendTemplateGenerationRemote,
		env.InstanaAgentKeyEnvRemote,
		env.DownloadKeyEnvRemote,
		env.InstanaAgentPodNameEnvRemote,
		env.PodIPEnvRemote,
	)
	// Create a map to track environment variables by name to ensure no duplicates
	// and to ensure pod.env values take precedence over agent.env values
	envVarMap := make(map[string]corev1.EnvVar)

	// Add base environment variables to the map
	for _, envVar := range baseEnvVars {
		envVarMap[envVar.Name] = envVar
	}

	// Add user-defined environment variables from the pod.env field
	// These will overwrite any existing variables with the same name
	for _, envVar := range d.InstanaAgentRemote.Spec.Agent.Pod.Env {
		envVarMap[envVar.Name] = envVar
	}

	// Convert the map back to a slice
	result := make([]corev1.EnvVar, 0, len(envVarMap))
	for _, envVar := range envVarMap {
		result = append(result, envVar)
	}

	// Sort the environment variables by name for consistency
	d.SortEnvVarsByName(result)
	return result
}

func (d *deploymentBuilder) getVolumes() ([]corev1.Volume, []corev1.VolumeMount) {
	return d.VolumeBuilderRemote.Build(
		volume.ConfigVolumeRemote,
		volume.TlsVolumeRemote,
		volume.RepoVolumeRemote,
	)
}

func (d *deploymentBuilder) getUserVolumes() ([]corev1.Volume, []corev1.VolumeMount) {
	return d.VolumeBuilderRemote.BuildFromUserConfig()
}

func (d *deploymentBuilder) getName() string {
	switch d.zone {
	case nil:
		return d.InstanaAgentRemote.Name
	default:
		return fmt.Sprintf("%s-%s", d.InstanaAgentRemote.Name, d.zone.Name.Name)
	}
}

func (d *deploymentBuilder) getHostName() string {
	if d.Spec.Hostname != nil && d.Spec.Hostname.Name != "" {
		// Sanitize hostname to make it Kubernetes-compatible
		hostname := sanitizeHostname(d.Spec.Hostname.Name)
		return hostname
	}
	return fmt.Sprintf("instana-agent-r-%s-%s", d.getName(), d.GetNamespace())
}

// sanitizeHostname ensures the hostname meets Kubernetes requirements:
// - Contains only lowercase alphanumeric characters or '-'
// - Starts with an alphanumeric character
// - Ends with an alphanumeric character
// - Is no longer than 63 characters
func sanitizeHostname(hostname string) string {
	// Convert to lowercase
	hostname = strings.ToLower(hostname)

	// Replace dots and other invalid characters with hyphens
	reg := regexp.MustCompile("[^a-z0-9-]")
	hostname = reg.ReplaceAllString(hostname, "-")

	// Ensure it starts with an alphanumeric character
	if len(hostname) > 0 && !regexp.MustCompile("^[a-z0-9]").MatchString(hostname) {
		hostname = "a" + hostname
	}

	// Ensure it ends with an alphanumeric character
	if len(hostname) > 0 && !regexp.MustCompile("[a-z0-9]$").MatchString(hostname) {
		hostname = hostname + "0"
	}

	// Truncate to 63 characters if needed
	if len(hostname) > 63 {
		hostname = hostname[:63]
		// After truncation, ensure it still ends with an alphanumeric character
		if !regexp.MustCompile("[a-z0-9]$").MatchString(hostname) {
			hostname = hostname[:62] + "0"
		}
	}

	return hostname
}

func (d *deploymentBuilder) getNonStandardLabels() map[string]string {
	switch d.zone {
	case nil:
		return nil
	default:
		return map[string]string{
			transformations.ZoneLabel: d.zone.Name.Name,
		}
	}
}

func (d *deploymentBuilder) getAffinity() *corev1.Affinity {
	switch d.zone {
	case nil:
		return &d.InstanaAgentRemote.Spec.Agent.Pod.Affinity
	default:
		return &d.zone.Affinity
	}
}

func (d *deploymentBuilder) getTolerations() []corev1.Toleration {
	switch d.zone {
	case nil:
		return d.InstanaAgentRemote.Spec.Agent.Pod.Tolerations
	default:
		return d.zone.Tolerations
	}
}

func (d *deploymentBuilder) build() *appsv1.Deployment {
	volumes, volumeMounts := d.getVolumes()
	userVolumes, userVolumeMounts := d.getUserVolumes()
	name := fmt.Sprintf("instana-agent-r-%s", d.getName())

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: d.Namespace,
			Labels:    d.getNonStandardLabels(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.To(int32(1)), // Set the number of replicas here
			Selector: &metav1.LabelSelector{
				MatchLabels: d.GetPodSelectorLabels(),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      d.getPodTemplateLabels(),
					Annotations: d.InstanaAgentRemote.Spec.Agent.Pod.Annotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "instana-agent-remote",
					Volumes:            append(volumes, userVolumes...),
					NodeSelector:       d.Spec.Agent.Pod.NodeSelector,
					PriorityClassName:  d.Spec.Agent.Pod.PriorityClassName,
					DNSPolicy:          corev1.DNSClusterFirst,
					ImagePullSecrets:   d.ImagePullSecrets(),
					Hostname:           d.getHostName(),
					Containers: []corev1.Container{
						{
							Name:            d.getName(),
							Image:           d.Spec.Agent.Image(),
							ImagePullPolicy: d.Spec.Agent.PullPolicy,
							VolumeMounts:    append(volumeMounts, userVolumeMounts...),
							Env:             d.getEnvVars(),
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"sh", "-c", "curl -f http://127.0.0.1:42699/status || exit 1",
										},
									},
								},
								InitialDelaySeconds: 600,
								TimeoutSeconds:      5,
								PeriodSeconds:       10,
								FailureThreshold:    3,
							},
							Resources: d.Spec.Agent.Pod.GetOrDefault(),
						},
					},
					Tolerations: d.getTolerations(),
					Affinity:    d.getAffinity(),
				},
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
			},
		},
	}
}

func (d *deploymentBuilder) Build() (res optional.Optional[client.Object]) {
	defer func() {
		res.IfPresent(
			func(dp client.Object) {
				d.statusManager.AddAgentDeployment(client.ObjectKeyFromObject(dp))
			},
		)
	}()

	switch {
	case d.Spec.Agent.Key == "" && d.Spec.Agent.KeysSecret == "":
		fallthrough
	case d.zone == nil && d.Spec.Zone.Name == "":
		fallthrough
	case d.zone != nil:
		return optional.Empty[client.Object]()
	default:
		return optional.Of[client.Object](d.build())
	}
}

func NewDeploymentBuilder(
	agent *instanav1.InstanaAgentRemote,
	statusManager status.InstanaAgentRemoteStatusManager,
	backend backends.RemoteSensorBackend,
	keysSecret *corev1.Secret,
) builder.ObjectBuilder {
	return &deploymentBuilder{
		InstanaAgentRemote:              agent,
		statusManager:                   statusManager,
		RemoteHelpers:                   helpers.NewRemoteHelpers(agent),
		PodSelectorLabelGeneratorRemote: transformations.PodSelectorLabelsRemote(agent, componentName),
		EnvBuilderRemote:                env.NewEnvBuilderRemote(agent, nil),
		VolumeBuilderRemote:             volume.NewVolumeBuilderRemote(agent),
		backend:                         backend,
		keysSecret:                      keysSecret,
	}
}
