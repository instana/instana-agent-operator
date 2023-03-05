package transformations

import (
	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
)

func TestAddCommonLabels(t *testing.T) {
	t.Run("with_empty_labels_initially", func(t *testing.T) {
		obj := v1.ConfigMap{}
		NewTransformations(&instanav1.InstanaAgent{
			ObjectMeta: metav1.ObjectMeta{
				Name: "asdf",
			},
		}).AddCommonLabels(&obj)

		assertions := require.New(t)

		assertions.Equal(map[string]string{
			"app.kubernetes.io/name":     "instana-agent",
			"app.kubernetes.io/instance": "asdf",
			"app.kubernetes.io/version":  "v0.0.0",
		}, obj.GetLabels())
	})
	t.Run("with_initial_labels", func(t *testing.T) {
		obj := v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"foo":   "bar",
					"hello": "world",
				},
			},
		}

		NewTransformations(&instanav1.InstanaAgent{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
		}).AddCommonLabels(&obj)

		assertions := require.New(t)

		assertions.Equal(map[string]string{
			"foo":                        "bar",
			"hello":                      "world",
			"app.kubernetes.io/name":     "instana-agent",
			"app.kubernetes.io/instance": "foo",
			"app.kubernetes.io/version":  "v0.0.0",
		}, obj.GetLabels())
	})
}
