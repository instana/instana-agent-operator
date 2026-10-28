/*
(c) Copyright IBM Corp. 2026
*/

package env

import (
	"testing"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/pointer"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestConfigPathValueFrom(t *testing.T) {
	for _, source := range []struct {
		name  string
		value *corev1.EnvVarSource
	}{
		{
			name: "configmap",
			value: &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "agent-config"},
				Key:                  "path",
				Optional:             pointer.To(true),
			}},
		},
		{
			name: "secret",
			value: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "agent-config"},
				Key:                  "path",
				Optional:             pointer.To(false),
			}},
		},
	} {
		t.Run(source.name, func(t *testing.T) {
			podEnv := []corev1.EnvVar{{Name: "CONFIG_PATH", ValueFrom: source.value}}
			spec := instanav1.BaseAgentSpec{Pod: instanav1.AgentPodSpec{Env: podEnv}}
			agent := &instanav1.InstanaAgent{Spec: instanav1.InstanaAgentSpec{Agent: spec}}
			remote := &instanav1.InstanaAgentRemote{
				Spec: instanav1.InstanaAgentRemoteSpec{Agent: spec},
			}
			expected := []corev1.EnvVar{
				{Name: "INSTANA_AGENT_CONFIG_PATH", ValueFrom: source.value.DeepCopy()},
			}
			for _, values := range [][]corev1.EnvVar{
				NewEnvBuilder(agent, nil).Build(InstanaAgentConfigPathEnv),
				NewEnvBuilderRemote(remote, nil).Build(InstanaAgentConfigPathEnvRemote),
			} {
				require.Equal(t, expected, values)
				require.NotSame(t, source.value, values[0].ValueFrom)
				if values[0].ValueFrom.ConfigMapKeyRef != nil {
					values[0].ValueFrom.ConfigMapKeyRef.Key = "changed"
				} else {
					values[0].ValueFrom.SecretKeyRef.Key = "changed"
				}
				require.Equal(t, expected[0].ValueFrom, source.value, "must not mutate the CR")
			}
		})
	}
}
