package transformations

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/pointer"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
)

func TestAddCommonLabels(t *testing.T) {
	t.Run(
		"with_empty_labels_initially", func(t *testing.T) {
			obj := v1.ConfigMap{}
			NewTransformations(
				&instanav1.InstanaAgent{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "asdf",
						Generation: 3,
					},
				},
			).AddCommonLabels(&obj, "eoisdijsdf")

			assertions := require.New(t)

			assertions.Equal(
				map[string]string{
					"app.kubernetes.io/name":       "instana-agent",
					"app.kubernetes.io/component":  "eoisdijsdf",
					"app.kubernetes.io/instance":   "asdf",
					"app.kubernetes.io/version":    "v0.0.0",
					"app.kubernetes.io/part-of":    "instana",
					"app.kubernetes.io/managed-by": "instana-agent-operator",
					"agent.instana.io/generation":  "v0.0.0-3",
				}, obj.GetLabels(),
			)
		},
	)
	t.Run(
		"with_initial_labels", func(t *testing.T) {
			obj := v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"foo":   "bar",
						"hello": "world",
					},
				},
			}

			NewTransformations(
				&instanav1.InstanaAgent{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "foo",
						Generation: 3,
					},
				},
			).AddCommonLabels(&obj, "roisoijdsf")

			assertions := require.New(t)

			assertions.Equal(
				map[string]string{
					"foo":                          "bar",
					"hello":                        "world",
					"app.kubernetes.io/name":       "instana-agent",
					"app.kubernetes.io/component":  "roisoijdsf",
					"app.kubernetes.io/instance":   "foo",
					"app.kubernetes.io/version":    "v0.0.0",
					"app.kubernetes.io/part-of":    "instana",
					"app.kubernetes.io/managed-by": "instana-agent-operator",
					"agent.instana.io/generation":  "v0.0.0-3",
				}, obj.GetLabels(),
			)
		},
	)
}

func TestAddOwnerReference(t *testing.T) {
	agent := instanav1.InstanaAgent{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "instana.io/v1",
			Kind:       "InstanaAgent",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "instana-agent",
			UID:  "iowegihsdgoijwefoih",
		},
	}
	t.Run(
		"with_no_previous_references", func(t *testing.T) {
			cm := v1.ConfigMap{}

			NewTransformations(&agent).AddOwnerReference(&cm)

			assertions := require.New(t)

			assertions.Equal(
				[]metav1.OwnerReference{
					{
						APIVersion:         "instana.io/v1",
						Kind:               "InstanaAgent",
						Name:               "instana-agent",
						UID:                "iowegihsdgoijwefoih",
						Controller:         pointer.To(true),
						BlockOwnerDeletion: pointer.To(true),
					},
				}, cm.OwnerReferences,
			)
		},
	)
	t.Run(
		"with_previous_references", func(t *testing.T) {
			otherOwner := metav1.OwnerReference{
				APIVersion:         "adsf",
				Kind:               "pojg",
				Name:               "ojregoi",
				UID:                "owjgepos",
				Controller:         pointer.To(false),
				BlockOwnerDeletion: pointer.To(false),
			}

			cm := v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{otherOwner},
				},
			}

			NewTransformations(&agent).AddOwnerReference(&cm)

			assertions := require.New(t)

			assertions.Equal(
				[]metav1.OwnerReference{
					otherOwner,
					{
						APIVersion:         "instana.io/v1",
						Kind:               "InstanaAgent",
						Name:               "instana-agent",
						UID:                "iowegihsdgoijwefoih",
						Controller:         pointer.To(true),
						BlockOwnerDeletion: pointer.To(true),
					},
				}, cm.OwnerReferences,
			)
		},
	)
}
