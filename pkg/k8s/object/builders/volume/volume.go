package volume

import (
	"github.com/instana/instana-agent-operator/pkg/optional"
	corev1 "k8s.io/api/core/v1"
)

type fromHostLiteralParams struct {
	name string
	path string
	*corev1.MountPropagationMode
}

func fromHostLiteral(params *fromHostLiteralParams) (optional.Builder[corev1.Volume], optional.Builder[corev1.VolumeMount]) { // TODO: Two return values for all or use container?
	return optional.BuilderFromLiteral(corev1.Volume{
			Name: params.name,
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: params.path,
				},
			},
		}),
		optional.BuilderFromLiteral(corev1.VolumeMount{
			Name:             params.name,
			MountPath:        params.path,
			MountPropagation: params.MountPropagationMode,
		})
}

// TODO
