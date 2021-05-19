/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package controllers

import (
	"context"

	instanaV1Beta1 "github.com/instana/instana-agent-operator/api/v1beta1"
	appV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// returns a Daemonset object with the data hold in instanaAgent crd instance
func newDaemonsetForCRD(crdInstance *instanaV1Beta1.InstanaAgent) *appV1.DaemonSet {
	//we need to have a same matched label for all our agent resources
	selectorLabels := buildLabels()
	podSpec := newPodSpec(crdInstance)
	return &appV1.DaemonSet{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      AppName,
			Namespace: AgentNameSpace,
			Labels:    selectorLabels,
		},
		Spec: appV1.DaemonSetSpec{
			Selector: &metaV1.LabelSelector{MatchLabels: selectorLabels},
			UpdateStrategy: appV1.DaemonSetUpdateStrategy{
				Type:          appV1.RollingUpdateDaemonSetStrategyType,
				RollingUpdate: &appV1.RollingUpdateDaemonSet{MaxUnavailable: &intstr.IntOrString{IntVal: 1}},
			},
			Template: coreV1.PodTemplateSpec{
				ObjectMeta: metaV1.ObjectMeta{
					Labels: selectorLabels,
				},
				Spec: podSpec,
			},
		},
	}
}

func newPodSpec(crdInstance *instanaV1Beta1.InstanaAgent) coreV1.PodSpec {
	trueVar := true
	secCtx := &coreV1.SecurityContext{
		Privileged: &trueVar,
	}

	AgentImageName := DefaultAgentImageName
	if crdInstance.Spec.Agent.Image != nil && len(crdInstance.Spec.Agent.Image.Name) > 0 {
		AgentImageName = crdInstance.Spec.Agent.Image.Name
	}

	envVars := buildEnvVars(crdInstance)

	return coreV1.PodSpec{
		ServiceAccountName: AgentServiceAccountName,
		HostIPC:            true,
		HostNetwork:        true,
		HostPID:            true,
		DNSPolicy:          coreV1.DNSClusterFirstWithHostNet,
		ImagePullSecrets:   []coreV1.LocalObjectReference{{Name: AgentImagePullSecretName}},
		Containers: []coreV1.Container{{
			Name:            AppName,
			Image:           AgentImageName,
			ImagePullPolicy: coreV1.PullAlways,
			Env:             envVars,
			SecurityContext: secCtx,
			Ports:           []coreV1.ContainerPort{{ContainerPort: AgentPort}},
			VolumeMounts:    buildVolumeMounts(crdInstance),
			LivenessProbe: &coreV1.Probe{
				InitialDelaySeconds: 300,
				TimeoutSeconds:      3,
				Handler: coreV1.Handler{
					HTTPGet: &coreV1.HTTPGetAction{
						Path: "/status",
						Port: intstr.FromInt(AgentPort),
					}}},
		}},
		Volumes:     buildVolumes(crdInstance),
		Tolerations: []coreV1.Toleration{},
	}
}
func buildEnvVars(crdInstance *instanaV1Beta1.InstanaAgent) []coreV1.EnvVar {
	userEnvVars := crdInstance.Spec.Agent.Env
	envVars := []coreV1.EnvVar{}
	for key, value := range userEnvVars {
		envVars = append(envVars, coreV1.EnvVar{Name: key, Value: value})
	}
	optional := true
	agentEnvVars := []coreV1.EnvVar{
		{Name: "INSTANA_OPERATOR_MANAGED", Value: "true"},
		{Name: "INSTANA_ZONE", Value: crdInstance.Spec.Zone.Name},
		{Name: "INSTANA_KUBERNETES_CLUSTER_NAME", Value: crdInstance.Spec.Cluster.Name},
		{Name: "INSTANA_AGENT_ENDPOINT", Value: crdInstance.Spec.Agent.EndpointHost},
		{Name: "INSTANA_AGENT_ENDPOINT_PORT", Value: crdInstance.Spec.Agent.EndpointPort},
		{Name: "INSTANA_AGENT_POD_NAME", ValueFrom: &coreV1.EnvVarSource{
			FieldRef: &coreV1.ObjectFieldSelector{
				FieldPath:  "metadata.name",
				APIVersion: "v1",
			},
		}},
		{Name: "POD_IP", ValueFrom: &coreV1.EnvVarSource{
			FieldRef: &coreV1.ObjectFieldSelector{
				FieldPath:  "status.podIP",
				APIVersion: "v1",
			},
		}},
		{Name: "INSTANA_AGENT_KEY", ValueFrom: &coreV1.EnvVarSource{
			SecretKeyRef: &coreV1.SecretKeySelector{
				LocalObjectReference: coreV1.LocalObjectReference{
					Name: AgentSecretName,
				},
				Key: "key",
			},
		}},
		{Name: "INSTANA_DOWNLOAD_KEY", ValueFrom: &coreV1.EnvVarSource{
			SecretKeyRef: &coreV1.SecretKeySelector{
				LocalObjectReference: coreV1.LocalObjectReference{
					Name: AgentSecretName,
				},
				Key:      "downloadKey",
				Optional: &optional,
			},
		}},
	}

	return append(agentEnvVars, envVars...)
}

func buildVolumeMounts(instance *instanaV1Beta1.InstanaAgent) []coreV1.VolumeMount {
	return []coreV1.VolumeMount{
		{
			Name:      "dev",
			MountPath: "/DEV",
		},
		{
			Name:      "run",
			MountPath: "/RUN",
		},
		{
			Name:      "var-run",
			MountPath: "/VAR/RUN",
		},
		{
			Name:      "var-run-kubo",
			MountPath: "/VAR/VCAP/SYS/RUN/DOCKER",
		},
		{
			Name:      "sys",
			MountPath: "/SYS",
		},
		{
			Name:      "var-log",
			MountPath: "/VAR/LOG",
		},
		{
			Name:      "var-lib",
			MountPath: "/VAR/LIB/CONTAINERS/STORAGE",
		},
		{
			Name:      "machine-id",
			MountPath: "/ETC/MACHINE-ID",
		},
		{
			Name:      "configuration",
			SubPath:   "configuration.yaml",
			MountPath: "/ROOT/configuration.yaml",
		},
	}
}

func buildVolumes(instance *instanaV1Beta1.InstanaAgent) []coreV1.Volume {
	return []coreV1.Volume{
		{
			Name: "dev",
			VolumeSource: coreV1.VolumeSource{
				HostPath: &coreV1.HostPathVolumeSource{
					Path: "/",
				},
			},
		},
		{
			Name: "run",
			VolumeSource: coreV1.VolumeSource{
				HostPath: &coreV1.HostPathVolumeSource{
					Path: "/run",
				},
			},
		},
		{
			Name: "var-run",
			VolumeSource: coreV1.VolumeSource{
				HostPath: &coreV1.HostPathVolumeSource{
					Path: "/var/run",
				},
			},
		},
		{
			Name: "var-run-kubo",
			VolumeSource: coreV1.VolumeSource{
				HostPath: &coreV1.HostPathVolumeSource{
					Path: "/var/vcap/sys/run/docker",
				},
			},
		},
		{
			Name: "sys",
			VolumeSource: coreV1.VolumeSource{
				HostPath: &coreV1.HostPathVolumeSource{
					Path: "/sys",
				},
			},
		},
		{
			Name: "var-log",
			VolumeSource: coreV1.VolumeSource{
				HostPath: &coreV1.HostPathVolumeSource{
					Path: "/var/log",
				},
			},
		},
		{
			Name: "var-lib",
			VolumeSource: coreV1.VolumeSource{
				HostPath: &coreV1.HostPathVolumeSource{
					Path: "/var/lib/containers/storage",
				},
			},
		},
		{
			Name: "machine-id",
			VolumeSource: coreV1.VolumeSource{
				HostPath: &coreV1.HostPathVolumeSource{
					Path: "/etc/machine-id",
				},
			},
		},
		{
			Name: "configuration",
			VolumeSource: coreV1.VolumeSource{
				ConfigMap: &coreV1.ConfigMapVolumeSource{LocalObjectReference: coreV1.LocalObjectReference{Name: AppName}},
			},
		},
	}
}

func (r *InstanaAgentReconciler) setDaemonsetReference(ctx context.Context, req ctrl.Request, crdInstance *instanaV1Beta1.InstanaAgent) error {
	daemonset := &appV1.DaemonSet{}
	err := r.Get(ctx, req.NamespacedName, daemonset)
	if err == nil {
		if err = controllerutil.SetControllerReference(crdInstance, daemonset, r.Scheme); err != nil {
			return err
		}
		if err = r.Update(ctx, daemonset); err != nil {
			r.Log.Error(err, "Failed to set controller reference for daemonset")
		}
		r.Log.Info("Set controller reference for daemonset was successfull")
	}
	return err
}
