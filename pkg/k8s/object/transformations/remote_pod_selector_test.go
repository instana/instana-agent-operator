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

package transformations

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
)

func assertRemoteSubset(assertions *require.Assertions, s map[string]string, subset map[string]string) {
	for k, v := range subset {
		assertions.Equal(v, s[k])
	}
}

func Test_podSelectorLabelGeneratorRemote_GetPodLabels(t *testing.T) {
	for _, test := range []struct {
		name            string
		userLabels      map[string]string
		otherAssertions func(assertions *require.Assertions, userLabels map[string]string, actual map[string]string)
	}{
		{
			name: "no_user_labels",
			otherAssertions: func(
				assertions *require.Assertions,
				userLabels map[string]string,
				actual map[string]string,
			) {
			},
		},
		{
			name: "contains_all_user_labels",
			userLabels: map[string]string{
				rand.String(10): rand.String(10),
				rand.String(10): rand.String(10),
				rand.String(10): rand.String(10),
			},
			otherAssertions: func(
				assertions *require.Assertions,
				userLabels map[string]string,
				actual map[string]string,
			) {
				assertRemoteSubset(assertions, actual, userLabels)
				assertions.Len(actual, len(userLabels)+5)
			},
		},
		{
			name: "overwrites_default_labels",
			userLabels: map[string]string{
				"app.kubernetes.io/name":       "abd",
				"app.kubernetes.io/instance":   "def",
				"app.kubernetes.io/component":  "ghi",
				"app.kubernetes.io/part-of":    "jkl",
				"app.kubernetes.io/managed-by": "mno",
			},
			otherAssertions: func(
				assertions *require.Assertions,
				userLabels map[string]string,
				actual map[string]string,
			) {
				assertions.Len(actual, 5)
			},
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				asssertions := require.New(t)

				agentName := rand.String(10)
				component := rand.String(10)

				agent := &instanav1.InstanaAgent{ObjectMeta: metav1.ObjectMeta{Name: agentName}}

				p := PodSelectorLabels(agent, component)

				actual := p.GetPodLabels(test.userLabels)

				assertRemoteSubset(
					asssertions, actual, map[string]string{
						"app.kubernetes.io/name":       "instana-agent",
						"app.kubernetes.io/instance":   agentName,
						"app.kubernetes.io/component":  component,
						"app.kubernetes.io/part-of":    "instana",
						"app.kubernetes.io/managed-by": "instana-agent-operator",
					},
				)
				asssertions.NotSame(&test.userLabels, &actual)

				test.otherAssertions(asssertions, test.userLabels, actual)
			},
		)
	}
}

func Test_podSelectorLabelGeneratorRemote_GetPodSelectorLabels(t *testing.T) {
	assertions := require.New(t)

	agentName := rand.String(10)
	component := rand.String(10)

	agent := &instanav1.RemoteAgent{ObjectMeta: metav1.ObjectMeta{Name: agentName}}

	p := PodSelectorLabelsRemote(agent, component)

	actual := p.GetPodSelectorLabels()
	assertions.Equal(
		map[string]string{
			"app.kubernetes.io/name":      "remote-instana-agent",
			"app.kubernetes.io/instance":  agentName,
			"app.kubernetes.io/component": component,
		}, actual,
	)
}
