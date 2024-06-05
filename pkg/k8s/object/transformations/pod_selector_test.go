package transformations

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
)

func assertSubset(assertions *require.Assertions, s map[string]string, subset map[string]string) {
	for k, v := range subset {
		assertions.Equal(v, s[k])
	}
}

func Test_podSelectorLabelGenerator_GetPodLabels(t *testing.T) {
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
				assertSubset(assertions, actual, userLabels)
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

				assertSubset(
					asssertions, actual, map[string]string{
						"app.kubernetes.io/name":       "instana-agent",
						"app.kubernetes.io/instance":   agentName,
						"app.kubernetes.io/component":  component,
						"app.kubernetes.io/part-of":    "instana",
						"app.kubernetes.io/managed-by": "instana-agent-operator",
					},
				)
				asssertions.NotSame(test.userLabels, actual)

				test.otherAssertions(asssertions, test.userLabels, actual)
			},
		)
	}
}

func Test_podSelectorLabelGenerator_GetPodSelectorLabels(t *testing.T) {
	assertions := require.New(t)

	agentName := rand.String(10)
	component := rand.String(10)

	agent := &instanav1.InstanaAgent{ObjectMeta: metav1.ObjectMeta{Name: agentName}}

	p := PodSelectorLabels(agent, component)

	actual := p.GetPodSelectorLabels()
	assertions.Equal(
		map[string]string{
			"app.kubernetes.io/name":      "instana-agent",
			"app.kubernetes.io/instance":  agentName,
			"app.kubernetes.io/component": component,
		}, actual,
	)
}
