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

package volume

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
)

const numDefinedVolumesRemote = 3

func rangeUntilRemote(n int) []RemoteVolume {
	res := make([]RemoteVolume, 0, n)

	for i := 0; i < n; i++ {
		res = append(res, RemoteVolume(i))
	}

	return res
}

func assertAllElementsUniqueRemote[T comparable](assertions *require.Assertions, list []T) {
	m := make(map[T]bool, len(list))

	for _, element := range list {
		m[element] = true
	}

	assertions.Equal(len(list), len(m))
}

func TestVolumeBuilderBuildsAreUniqueRemote(t *testing.T) {
	t.Run(
		"each returned volume and volume mount is unique", func(t *testing.T) {
			assertions := require.New(t)

			vb := NewVolumeBuilderRemote(&instanav1.RemoteAgent{})
			volume, volumeMount := vb.Build(rangeUntilRemote(numDefinedVolumesRemote)...)

			assertions.Len(volume, numDefinedVolumesRemote-2)
			assertions.Len(volumeMount, numDefinedVolumesRemote-2)
			assertAllElementsUniqueRemote(assertions, volume)
			assertAllElementsUniqueRemote(assertions, volumeMount)
		},
	)

}

func TestVolumeBuilderPanicsWhenVolumeNumberDoesntExistRemote(t *testing.T) {
	t.Run(
		"panics once a volume is introduced that isn't found in the defined volumes", func(t *testing.T) {
			assert.PanicsWithError(t, "unknown volume requested", func() {
				_, _ = NewVolumeBuilderRemote(&instanav1.RemoteAgent{}).
					Build([]RemoteVolume{RemoteVolume(9999)}...)
			})
		},
	)
}

func TestVolumeBuilderBuildRemote(t *testing.T) {
	for _, test := range []struct {
		name               string
		expectedNumVolumes int
	}{
		{
			name:               "testVolumes",
			expectedNumVolumes: 1,
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)

				vb := NewVolumeBuilderRemote(&instanav1.RemoteAgent{})

				volumes, volumeMounts := vb.Build(rangeUntilRemote(numDefinedVolumesRemote)...)

				assertions.Len(volumes, test.expectedNumVolumes)
				assertions.Len(volumeMounts, test.expectedNumVolumes)
			},
		)
	}
}

func TestVolumeBuilderBuildFromUserConfigRemote(t *testing.T) {
	assertions := require.New(t)
	volumeName := "testVolume"
	agent := &instanav1.InstanaAgent{
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				Pod: instanav1.AgentPodSpec{
					Volumes: []corev1.Volume{
						{
							Name: volumeName,
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name: volumeName,
						},
					},
				},
			},
		},
	}
	for _, test := range []struct {
		name string
	}{
		{
			name: "UserConfig",
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				vb := NewVolumeBuilder(agent, true)
				volumes, volumeMounts := vb.BuildFromUserConfig()
				assertions.Equal(volumeName, volumes[0].Name)
				assertions.Equal(volumeName, volumeMounts[0].Name)
			},
		)
	}
}

func TestVolumeBuilderTlsSpecRemote(t *testing.T) {
	for _, test := range []struct {
		name               string
		volume             RemoteVolume
		volumeName         string
		instanaAgent       instanav1.RemoteAgent
		expectedNumVolumes int
	}{
		{
			name:       "Should return an TLS volume when Agent configuration has TLS Spec values",
			volume:     TlsVolumeRemote,
			volumeName: "remote-agent-tls",
			instanaAgent: instanav1.RemoteAgent{
				Spec: instanav1.RemoteAgentSpec{
					Agent: instanav1.BaseAgentSpec{
						TlsSpec: instanav1.TlsSpec{
							SecretName: "very-secret",
						},
					},
				},
			},
			expectedNumVolumes: 1,
		},
		{
			name:               "Should not return a TLS volume entry when TLS Spec values are missing from Agent configuration",
			volume:             TlsVolumeRemote,
			instanaAgent:       instanav1.RemoteAgent{Spec: instanav1.RemoteAgentSpec{Agent: instanav1.BaseAgentSpec{TlsSpec: instanav1.TlsSpec{}}}},
			expectedNumVolumes: 0,
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)

				vb := NewVolumeBuilderRemote(&test.instanaAgent)

				actualvolumes, actualVolumeMounts := vb.Build(test.volume)
				assertions.Len(actualvolumes, test.expectedNumVolumes)
				assertions.Len(actualVolumeMounts, test.expectedNumVolumes)

				if len(actualvolumes) > 0 && test.volumeName != "" {
					assertions.Equal(actualvolumes[0].Name, test.volumeName)
				}
			},
		)
	}
}

func TestVolumeBuilderRepositoryRemote(t *testing.T) {
	t.Run(
		"Build returns a Repository struct when the Repository exists", func(t *testing.T) {
			assertions := require.New(t)

			vb := NewVolumeBuilderRemote(
				&instanav1.RemoteAgent{
					Spec: instanav1.RemoteAgentSpec{
						Agent: instanav1.BaseAgentSpec{
							Host: instanav1.HostSpec{
								Repository: "very-repository",
							},
						},
					},
				})

			actualvolumes, actualVolumeMounts := vb.Build(RepoVolumeRemote)

			assertions.Len(actualvolumes, 1)
			assertions.Len(actualVolumeMounts, 1)
		},
	)
}
