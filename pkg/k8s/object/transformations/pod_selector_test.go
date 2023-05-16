package transformations

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
)

// TODO

// func Test_podSelectorLabelGenerator_GetPodLabels(t *testing.T) {
// 	asssertions := require.New(t)
//
// 	agentName := rand.String(10)
// 	component := rand.String(10)
//
// 	agent := &instanav1.InstanaAgent{ObjectMeta: metav1.ObjectMeta{Name: agentName}}
//
// 	userLabels := map[string]string{
// 		rand.String(10): rand.String(10),
// 		rand.String(10): rand.String(10),
// 		rand.String(10): rand.String(10),
// 	}
//
// 	p := PodSelectorLabels(agent, component)
//
// 	actual := p.GetPodLabels(userLabels)
//
// 	asssertions.Contains(actual, userLabels)
// 	asssertions.Contains(
// 		actual, map[string]string{
// 			"app.kubernetes.io/name":       "instana-agent",
// 			"app.kubernetes.io/instance":   agentName,
// 			"app.kubernetes.io/component":  component,
// 			"app.kubernetes.io/part-of":    "instana",
// 			"app.kubernetes.io/managed-by": "instana-agent-operator",
// 		},
// 	)
// }

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
