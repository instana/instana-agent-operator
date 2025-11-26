/*
(c) Copyright IBM Corp. 2024,2025
*/

package deployment

import (
	"crypto/sha256"
	"fmt"
	"path"
	"regexp"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	backends "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/backends"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/env"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/ports"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/volume"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/transformations"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/status"
	"github.com/instana/instana-agent-operator/pkg/map_defaulter"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

const componentName = constants.ComponentK8Sensor

// DeploymentContext holds additional context for the deployment
type DeploymentContext struct {
	DiscoveredETCDTargets       []string
	ETCDCASecretName            string
	OpenShiftETCDResourcesExist bool
}

type deploymentBuilder struct {
	*instanav1.InstanaAgent
	statusManager status.AgentStatusManager

	helpers helpers.Helpers
	transformations.PodSelectorLabelGenerator
	env.EnvBuilder
	volume.VolumeBuilder
	ports.PortsBuilder
	backend           backends.K8SensorBackend
	keysSecret        *corev1.Secret
	deploymentContext *DeploymentContext
	isOpenShift       bool
}

func (d *deploymentBuilder) IsNamespaced() bool {
	return true
}

func (d *deploymentBuilder) ComponentName() string {
	return componentName
}

func (d *deploymentBuilder) getPodTemplateLabels() map[string]string {
	podLabels := optional.Of(d.Spec.Agent.Pod.Labels).GetOrDefault(make(map[string]string, 3))
	podLabels[constants.LabelAgentMode] = string(instanav1.KUBERNETES)
	return d.GetPodLabels(podLabels)
}

func (d *deploymentBuilder) getEnvVars() []corev1.EnvVar {
	// Build common env vars - exclude ETCDCAFileEnv for OpenShift as it's set in OpenShift-specific section
	envVarsToInclude := []env.EnvVar{
		env.BackendURLEnv,
		env.AgentZoneEnv,
		env.PodUIDEnv,
		env.PodNamespaceEnv,
		env.PodNameEnv,
		env.PodIPEnv,
		env.NoProxyEnv,
		env.RedactK8sSecretsEnv,
		env.ConfigPathEnv,
		env.ETCDInsecureEnv,
		env.ETCDTargetsEnv,
		env.ControlPlaneCAFileEnv,
		env.RestClientHostAllowlistEnv,
	}

	// For vanilla Kubernetes, include ETCDCAFileEnv (for custom CA configuration)
	// For OpenShift, skip it here as it's added in the OpenShift-specific section below
	if !d.isOpenShift {
		envVarsToInclude = append(envVarsToInclude, env.ETCDCAFileEnv)
	}

	envVars := d.EnvBuilder.Build(envVarsToInclude...)

	// Add OpenShift-specific environment variables
	if d.isOpenShift {
		// Only add ETCD configuration if resources are available
		if d.deploymentContext != nil && d.deploymentContext.OpenShiftETCDResourcesExist {
			envVars = append(envVars, []corev1.EnvVar{
				{
					Name:  constants.EnvETCDMetricsURL,
					Value: constants.GetETCDOCPMetricsURL(),
				},
				{
					Name:  constants.EnvETCDRequestTimeout,
					Value: "15s",
				},
				{
					Name:  constants.EnvETCDCAFile,
					Value: path.Join(constants.ETCDMetricsCAMountPath, constants.ETCDCABundleFileName),
				},
				{
					Name:  constants.EnvETCDCertFile,
					Value: path.Join(constants.ETCDClientCertMountPath, constants.ETCDClientCertFileName),
				},
				{
					Name:  constants.EnvETCDKeyFile,
					Value: path.Join(constants.ETCDClientCertMountPath, constants.ETCDClientKeyFileName),
				},
			}...)
		}
	} else {
		// Add discovered ETCD targets for vanilla Kubernetes
		if d.deploymentContext != nil && len(d.deploymentContext.DiscoveredETCDTargets) > 0 {
			// Only add if not already specified in the CR
			if len(d.Spec.K8sSensor.ETCD.Targets) == 0 {
				envVars = append(envVars, corev1.EnvVar{
					Name:  constants.EnvETCDTargets,
					Value: strings.Join(d.deploymentContext.DiscoveredETCDTargets, ","),
				})

				// Add CA file env var if CA secret is available
				if d.deploymentContext.ETCDCASecretName != "" {
					envVars = append(envVars, corev1.EnvVar{
						Name:  constants.EnvETCDCAFile,
						Value: constants.ETCDCAMountPath + "/ca.crt",
					})
				}
			}
		}
	}
	backendEnvVars := []corev1.EnvVar{
		{
			Name: "BACKEND",
			ValueFrom: &corev1.EnvVarSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: d.helpers.K8sSensorResourcesName(),
					},
					Key: constants.BackendKey + d.backend.ResourceSuffix,
				},
			},
		},
	}

	// Only add the AGENT_KEY and HTTPS_PROXY environment variable if secret mounts are explicitly disabled
	if d.Spec.UseSecretMounts != nil && !*d.Spec.UseSecretMounts {
		// For all backends, use the key from the secret
		backendEnvVars = append(backendEnvVars, d.getAgentKeyEnvVar())
		backendEnvVars = append(backendEnvVars, d.EnvBuilder.Build(env.HTTPSProxyEnv)...)
	}

	envVars = append(backendEnvVars, envVars...)
	d.helpers.SortEnvVarsByName(envVars)
	return envVars
}

func (d *deploymentBuilder) getVolumes() ([]corev1.Volume, []corev1.VolumeMount) {
	volumesToBuild := []volume.Volume{
		volume.ConfigVolume,
		volume.ETCDCAVolume, // Always include for custom ETCD CA configuration
		volume.ControlPlaneCAVolume,
	}

	// Add ETCD client certificate volume for OpenShift if resources exist
	if d.isOpenShift &&
		d.deploymentContext != nil &&
		d.deploymentContext.OpenShiftETCDResourcesExist {
		volumesToBuild = append(volumesToBuild, volume.ETCDClientCertVolume)
	}

	// Add secrets volume if secret mounts are enabled (default behavior)
	if d.Spec.UseSecretMounts == nil || *d.Spec.UseSecretMounts {
		volumesToBuild = append(volumesToBuild, volume.K8SensorSecretsVolume)
	}

	volumes, mounts := d.VolumeBuilder.Build(volumesToBuild...)

	// Add CA cert if available from discovery
	if d.deploymentContext != nil && d.deploymentContext.ETCDCASecretName != "" && len(d.Spec.K8sSensor.ETCD.Targets) == 0 {
		volumes = append(volumes, corev1.Volume{
			Name: "etcd-ca",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{ // pragma: allowlist secret
					SecretName: d.deploymentContext.ETCDCASecretName,
					Items: []corev1.KeyToPath{
						{
							Key:  "ca.crt",
							Path: "ca.crt",
						},
					},
				},
			},
		})

		mounts = append(mounts, corev1.VolumeMount{
			Name:      "etcd-ca",
			MountPath: "/var/run/secrets/etcd",
			ReadOnly:  true,
		})
	}

	return volumes, mounts
}

// getAgentKeyEnvVar returns an environment variable for the AGENT_KEY that references the key from a secret
// It works for both the main backend and additional backends by using the appropriate key suffix
func (d *deploymentBuilder) getAgentKeyEnvVar() corev1.EnvVar {
	return corev1.EnvVar{
		Name: "AGENT_KEY",
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: optional.Of(d.Spec.Agent.KeysSecret).GetOrDefault(d.Name),
				},
				Key: constants.AgentKey + d.backend.ResourceSuffix,
			},
		},
	}
}

// K8Sensor relies on this label for internal sharding logic for some reason, if you remove it the k8sensor will break
func addAppLabel(labels map[string]string) map[string]string {
	labelsDefaulter := map_defaulter.NewMapDefaulter(&labels)
	labelsDefaulter.SetIfEmpty("app", "k8sensor")
	return labels
}

func (d *deploymentBuilder) getPodAnnotationsWithBackendChecksum() map[string]string {
	// Deep copy annotations to extend them with a checksum
	annotations := make(map[string]string, len(d.Spec.Agent.Pod.Annotations)+1)
	for k, v := range d.Spec.Agent.Pod.Annotations {
		annotations[k] = v
	}

	h := sha256.New()
	if d.Spec.Agent.KeysSecret != "" {
		h.Write([]byte(d.backend.EndpointHost + d.backend.EndpointPort))
		// keysSecret contains the relevant data, if no key is found ignore it for the checksum
		if d.keysSecret != nil {
			endpointKeyFromSecret := d.keysSecret.Data["key"+d.backend.ResourceSuffix]
			h.Write(endpointKeyFromSecret)
		}
	} else {
		// backend secret was part of the CR
		h.Write([]byte(d.backend.EndpointHost + d.backend.EndpointPort + d.backend.EndpointKey))
	}

	annotations["checksum/backend"] = fmt.Sprintf("%x", h.Sum(nil))
	return annotations
}

func (d *deploymentBuilder) build() *appsv1.Deployment {
	volumes, mounts := d.getVolumes()

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.helpers.K8sSensorResourcesName() + d.backend.ResourceSuffix,
			Namespace: d.Namespace,
			Labels:    addAppLabel(nil),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas:        pointer.To(int32(d.Spec.K8sSensor.DeploymentSpec.Replicas)),
			MinReadySeconds: int32(d.Spec.K8sSensor.DeploymentSpec.MinReadySeconds),
			Selector: &metav1.LabelSelector{
				MatchLabels: addAppLabel(d.GetPodSelectorLabels()),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      addAppLabel(d.getPodTemplateLabels()),
					Annotations: d.getPodAnnotationsWithBackendChecksum(),
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: d.helpers.K8sSensorResourcesName(),
					NodeSelector:       d.Spec.K8sSensor.DeploymentSpec.Pod.NodeSelector,
					PriorityClassName:  d.Spec.K8sSensor.DeploymentSpec.Pod.PriorityClassName,
					ImagePullSecrets:   d.helpers.ImagePullSecrets(),
					Containers: []corev1.Container{
						{
							Name:            "instana-agent",
							Image:           d.Spec.K8sSensor.ImageSpec.Image(),
							ImagePullPolicy: d.Spec.K8sSensor.ImageSpec.PullPolicy,
							Command:         []string{"/ko-app/k8sensor"},
							Args:            d.getK8SensorArgs(),
							Env:             d.getEnvVars(),
							VolumeMounts:    mounts,
							Resources:       d.Spec.K8sSensor.DeploymentSpec.Pod.ResourceRequirements.GetOrDefault(),
							Ports: []corev1.ContainerPort{
								ports.InstanaAgentAPIPortConfig.AsContainerPort(),
							},
						},
					},
					// k8sensor is run as a "k8sensor" user (i.e: uid 1000), and thus reading the files from the secret volume
					// requires setting FSGroup to 1000
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup: pointer.To(int64(1000)),
					},
					Volumes:     volumes,
					Tolerations: d.Spec.K8sSensor.DeploymentSpec.Pod.Tolerations,
					Affinity: pointer.To(
						optional.Of(d.Spec.K8sSensor.DeploymentSpec.Pod.Affinity).GetOrDefault(
							corev1.Affinity{
								PodAntiAffinity: &corev1.PodAntiAffinity{
									PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
										{
											Weight: 100,
											PodAffinityTerm: corev1.PodAffinityTerm{
												LabelSelector: &metav1.LabelSelector{
													MatchExpressions: []metav1.LabelSelectorRequirement{
														{
															Key:      constants.LabelAgentMode,
															Operator: metav1.LabelSelectorOpIn,
															Values: []string{
																string(instanav1.KUBERNETES),
															},
														},
													},
												},
												TopologyKey: corev1.LabelHostname,
											},
										},
									},
								},
							},
						),
					),
				},
			},
		},
	}
}

func (d *deploymentBuilder) Build() (res optional.Optional[client.Object]) {
	defer func() {
		res.IfPresent(
			func(dpl client.Object) {
				d.statusManager.SetK8sSensorDeployment(client.ObjectKeyFromObject(dpl))
			},
		)
	}()

	switch (d.Spec.Agent.Key == "" && d.Spec.Agent.KeysSecret == "") ||
		(d.Spec.Zone.Name == "" && d.Spec.Cluster.Name == "") ||
		!pointer.DerefOrDefault(d.Spec.K8sSensor.DeploymentSpec.Enabled.Enabled, true) {
	case true:
		return optional.Empty[client.Object]()
	default:
		return optional.Of[client.Object](d.build())
	}
}

func NewDeploymentBuilder(
	agent *instanav1.InstanaAgent,
	isOpenShift bool,
	statusManager status.AgentStatusManager,
	backend backends.K8SensorBackend,
	keysSecret *corev1.Secret,
	deploymentContext *DeploymentContext,
) builder.ObjectBuilder {
	return &deploymentBuilder{
		InstanaAgent:              agent,
		statusManager:             statusManager,
		helpers:                   helpers.NewHelpers(agent),
		PodSelectorLabelGenerator: transformations.PodSelectorLabels(agent, componentName),
		EnvBuilder:                env.NewEnvBuilder(agent, nil),
		VolumeBuilder: volume.NewVolumeBuilder(agent, isOpenShift).
			WithBackendResourceSuffix(backend.ResourceSuffix),
		PortsBuilder:      ports.NewPortsBuilder(agent.Spec.OpenTelemetry),
		backend:           backend,
		keysSecret:        keysSecret,
		deploymentContext: deploymentContext,
		isOpenShift:       isOpenShift,
	}
}

const (
	// PollRateRegex defines the validation pattern for pollrate values (seconds only)
	PollRateRegex = `^[0-9]+s$`
	// DefaultPollRate is the default polling rate for k8sensor
	DefaultPollRate = "10s"
)

// validatePollRate checks if the pollrate matches the same pattern as the CRD validation
func (d *deploymentBuilder) validatePollRate(pollRate string) bool {
	matched, err := regexp.MatchString(PollRateRegex, pollRate)
	if err != nil {
		return false
	}
	return matched
}

// getK8SensorArgs returns the command line arguments for the k8sensor
func (d *deploymentBuilder) getK8SensorArgs() []string {
	pollRate := DefaultPollRate
	if d.Spec.K8sSensor.PollRate != "" {
		if d.validatePollRate(d.Spec.K8sSensor.PollRate) {
			pollRate = d.Spec.K8sSensor.PollRate
		}
	}

	args := []string{"-pollrate", pollRate}

	if d.Spec.UseSecretMounts == nil || *d.Spec.UseSecretMounts {
		// Use backend-specific secret file key to support multiple backends
		agentKeyFile := constants.SecretFileAgentKey + d.backend.ResourceSuffix
		args = append(args,
			"-agent-key-file",
			fmt.Sprintf("%s/%s", constants.InstanaSecretsDirectory, agentKeyFile))

		// Add HTTPS_PROXY file argument if proxy host is configured
		if d.Spec.Agent.ProxyHost != "" {
			args = append(
				args,
				"-https-proxy-file",
				fmt.Sprintf(
					"%s/%s",
					constants.InstanaSecretsDirectory,
					constants.SecretFileHttpsProxy,
				),
			)
		}
	}

	return args
}
