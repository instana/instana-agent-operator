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
				"agent.instana.io/generation":  "4",
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
				"agent.instana.io/generation":  "3",
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
		{
			name: "with_duplicate_ref",
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

func TestTransformations_AddOwnerReference_WithSameNameDifferentUID(t *testing.T) {
	assertions := require.New(t)

	// existing ConfigMap with a stale owner ref (same name, old UID)
	configMap := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "instana.io/v1",
					Kind:               "InstanaAgent",
					Name:               "instana-agent",
					UID:                "old-uid-value", // Different (stale) UID
					Controller:         pointer.To(true),
					BlockOwnerDeletion: pointer.To(true),
				},
				{
					APIVersion: "other-api/v1",
					Kind:       "OtherKind",
					Name:       "other-name",
					UID:        "other-uid",
					// Controller false for unrelated owner
				},
			},
		},
	}

	// Set the transformations' OwnerReference to the new (recreated) CR UID
	tform := &transformations{
		OwnerReference: metav1.OwnerReference{
			APIVersion:         "instana.io/v1",
			Kind:               "InstanaAgent",
			Name:               "instana-agent",
			UID:                "new-uid-value", // new UID (recreated CR)
			Controller:         pointer.To(true),
			BlockOwnerDeletion: pointer.To(true),
		},
	}

	// Call our corrected AddOwnerReference implementation
	tform.AddOwnerReference(&configMap)

	// Now the configMap should have two owner references:
	// - the unrelated "other-name" ref (unchanged)
	// - the new "instana-agent" ref (new UID), and NOT the old one
	refs := configMap.GetOwnerReferences()
	// Build a map for easy assertions
	refByName := map[string]metav1.OwnerReference{}
	for _, r := range refs {
		refByName[r.Name] = r
	}

	// Expect unrelated owner to still exist
	other, ok := refByName["other-name"]
	assertions.True(ok, "other-name owner reference should be preserved")
	assertions.Equal("other-uid", string(other.UID))

	// Expect instana-agent to exist and have the new UID
	instana, ok := refByName["instana-agent"]
	assertions.True(ok, "instana-agent owner reference should exist")
	assertions.Equal("new-uid-value", string(instana.UID))

	// Ensure the stale UID is not present
	for _, r := range refs {
		assertions.NotEqual(
			"old-uid-value",
			string(r.UID),
			"stale owner reference should be removed",
		)
	}
}
