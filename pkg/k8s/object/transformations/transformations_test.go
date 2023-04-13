package transformations

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/pointer"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
)

func TestTransformations_AddCommonLabels(t *testing.T) {
	for _, tc := range []struct {
		name           string
		initialLabels  map[string]string
		instanaAgent   instanav1.InstanaAgent
		componentLabel string
		versionEnv     string
		expectedLabels map[string]string
	}{
		{
			name:          "with_empty_labels_initially",
			initialLabels: nil,
			instanaAgent: instanav1.InstanaAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "asdf",
					Generation: 4,
				},
			},
			componentLabel: "eoisdijsdf",
			versionEnv:     "v0.0.1",
			expectedLabels: map[string]string{
				"app.kubernetes.io/name":       "instana-agent",
				"app.kubernetes.io/component":  "eoisdijsdf",
				"app.kubernetes.io/instance":   "asdf",
				"app.kubernetes.io/version":    "v0.0.1",
				"app.kubernetes.io/part-of":    "instana",
				"app.kubernetes.io/managed-by": "instana-agent-operator",
				"agent.instana.io/generation":  "v0.0.1-4",
			},
		},
		{
			name: "with_initial_labels",
			initialLabels: map[string]string{
				"foo":   "bar",
				"hello": "world",
			},
			instanaAgent: instanav1.InstanaAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "yrsthsdht",
					Generation: 3,
				},
			},
			componentLabel: "roisoijdsf",
			versionEnv:     "v0.0.2",
			expectedLabels: map[string]string{
				"foo":                          "bar",
				"hello":                        "world",
				"app.kubernetes.io/name":       "instana-agent",
				"app.kubernetes.io/component":  "roisoijdsf",
				"app.kubernetes.io/instance":   "yrsthsdht",
				"app.kubernetes.io/version":    "v0.0.2",
				"app.kubernetes.io/part-of":    "instana",
				"app.kubernetes.io/managed-by": "instana-agent-operator",
				"agent.instana.io/generation":  "v0.0.2-3",
			},
		},
	} {
		t.Run(
			tc.name, func(t *testing.T) {
				assertions := require.New(t)

				version = tc.versionEnv

				obj := v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Labels: tc.initialLabels,
					},
				}

				NewTransformations(&tc.instanaAgent).AddCommonLabels(&obj, tc.componentLabel)

				assertions.Equal(tc.expectedLabels, obj.GetLabels())
			},
		)
	}
}

func TestTransformations_AddOwnerReference(t *testing.T) {
	for _, tc := range []struct {
		name         string
		configMap    v1.ConfigMap
		expectedRefs []metav1.OwnerReference
	}{
		{
			name:      "with_no_previous_references",
			configMap: v1.ConfigMap{},
			expectedRefs: []metav1.OwnerReference{
				{
					APIVersion:         "instana.io/v1",
					Kind:               "InstanaAgent",
					Name:               "instana-agent",
					UID:                "iowegihsdgoijwefoih",
					Controller:         pointer.To(true),
					BlockOwnerDeletion: pointer.To(true),
				},
			},
		},
		{
			name: "with_previous_references",
			configMap: v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "adsf",
							Kind:               "pojg",
							Name:               "ojregoi",
							UID:                "owjgepos",
							Controller:         pointer.To(false),
							BlockOwnerDeletion: pointer.To(false),
						},
					},
				},
			},
			expectedRefs: []metav1.OwnerReference{
				{
					APIVersion:         "adsf",
					Kind:               "pojg",
					Name:               "ojregoi",
					UID:                "owjgepos",
					Controller:         pointer.To(false),
					BlockOwnerDeletion: pointer.To(false),
				},
				{
					APIVersion:         "instana.io/v1",
					Kind:               "InstanaAgent",
					Name:               "instana-agent",
					UID:                "iowegihsdgoijwefoih",
					Controller:         pointer.To(true),
					BlockOwnerDeletion: pointer.To(true),
				},
			},
		},
	} {
		t.Run(
			tc.name, func(t *testing.T) {
				assertions := require.New(t)

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

				NewTransformations(&agent).AddOwnerReference(&tc.configMap)

				assertions.Equal(tc.expectedRefs, tc.configMap.OwnerReferences)
			},
		)
	}
}
