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

package volume

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
)

const numDefinedVolumes = 14

func rangeUntil(n int) []Volume {
	res := make([]Volume, 0, n)

	for i := 0; i < n; i++ {
		res = append(res, Volume(i))
	}

	return res
}

func assertAllElementsUnique[T comparable](assertions *require.Assertions, list []T) {
	m := make(map[T]bool, len(list))

	for _, element := range list {
		m[element] = true
	}

	assertions.Equal(len(list), len(m))
}

func TestVolumeBuilderBuildsAreUnique(t *testing.T) {
	t.Run(
		"each returned volume and volume mount is unique", func(t *testing.T) {
			assertions := require.New(t)

			vb := NewVolumeBuilder(&instanav1.InstanaAgent{}, false)
			volume, volumeMount := vb.Build(rangeUntil(numDefinedVolumes)...)

			assertions.Len(volume, numDefinedVolumes-2)
			assertions.Len(volumeMount, numDefinedVolumes-2)
			assertAllElementsUnique(assertions, volume)
			assertAllElementsUnique(assertions, volumeMount)
		},
	)

}

func TestVolumeBuilderPanicsWhenVolumeNumberDoesntExist(t *testing.T) {
	t.Run(
		"panics once a volume is introduced that isn't found in the defined volumes", func(t *testing.T) {
			assert.PanicsWithError(t, "unknown volume requested", func() {
				_, _ = NewVolumeBuilder(&instanav1.InstanaAgent{}, false).
					Build([]Volume{Volume(9999)}...)
			})
		},
	)
}

func TestVolumeBuilderBuild(t *testing.T) {
	for _, test := range []struct {
		name               string
		isOpenShift        bool
		expectedNumVolumes int
	}{
		{
			name:               "isOpenShift",
			isOpenShift:        true,
			expectedNumVolumes: 9,
		},
		{
			name:               "isNotOpenShift",
			isOpenShift:        false,
			expectedNumVolumes: 12,
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)

				vb := NewVolumeBuilder(&instanav1.InstanaAgent{}, test.isOpenShift)

				volumes, volumeMounts := vb.Build(rangeUntil(numDefinedVolumes)...)

				assertions.Len(volumes, test.expectedNumVolumes)
				assertions.Len(volumeMounts, test.expectedNumVolumes)
			},
		)
	}
}

func TestVolumeBuilderBuildFromUserConfig(t *testing.T) {
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
		name        string
		isOpenShift bool
	}{
		{
			name:        "isOpenShift",
			isOpenShift: true,
		},
		{
			name:        "isNotOpenShift",
			isOpenShift: false,
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

func TestVolumeBuilderTlsSpec(t *testing.T) {
	for _, test := range []struct {
		name               string
		volume             Volume
		volumeName         string
		instanaAgent       instanav1.InstanaAgent
		expectedNumVolumes int
	}{
		{
			name:       "Should return an TLS volume when Agent configuration has TLS Spec values",
			volume:     TlsVolume,
			volumeName: "instana-agent-tls",
			instanaAgent: instanav1.InstanaAgent{
				Spec: instanav1.InstanaAgentSpec{
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
			volume:             TlsVolume,
			instanaAgent:       instanav1.InstanaAgent{Spec: instanav1.InstanaAgentSpec{Agent: instanav1.BaseAgentSpec{TlsSpec: instanav1.TlsSpec{}}}},
			expectedNumVolumes: 0,
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)

				vb := NewVolumeBuilder(&test.instanaAgent, false)

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

func TestVolumeBuilderRepository(t *testing.T) {
	t.Run(
		"Build returns a Repository struct when the Repository exists", func(t *testing.T) {
			assertions := require.New(t)

			vb := NewVolumeBuilder(
				&instanav1.InstanaAgent{
					Spec: instanav1.InstanaAgentSpec{
						Agent: instanav1.BaseAgentSpec{
							Host: instanav1.HostSpec{
								Repository: "very-repository",
							},
						},
					},
				}, false)

			actualvolumes, actualVolumeMounts := vb.Build(RepoVolume)

			assertions.Len(actualvolumes, 1)
			assertions.Len(actualVolumeMounts, 1)
		},
	)
}
