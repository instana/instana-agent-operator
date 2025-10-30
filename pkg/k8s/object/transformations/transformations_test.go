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
	// This test verifies that when an owner reference with the same name but different UID exists,
	// it gets replaced with the new one
	assertions := require.New(t)

	// Create a ConfigMap with an existing owner reference
	configMap := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "instana.io/v1",
					Kind:               "InstanaAgent",
					Name:               "instana-agent",
					UID:                "old-uid-value", // Different UID
					Controller:         pointer.To(true),
					BlockOwnerDeletion: pointer.To(true),
				},
				{
					APIVersion:         "other-api/v1",
					Kind:               "OtherKind",
					Name:               "other-name",
					UID:                "other-uid",
					Controller:         pointer.To(false),
					BlockOwnerDeletion: pointer.To(false),
				},
			},
		},
	}

	// Create a new agent with the same name but different UID
	agent := instanav1.InstanaAgent{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "instana.io/v1",
			Kind:       "InstanaAgent",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "instana-agent",
			UID:  "new-uid-value", // New UID
		},
	}

	// Add the owner reference
	NewTransformations(&agent).AddOwnerReference(&configMap)

	// Verify that the old reference was replaced and the unrelated one was kept
	expectedRefs := []metav1.OwnerReference{
		{
			APIVersion:         "other-api/v1",
			Kind:               "OtherKind",
			Name:               "other-name",
			UID:                "other-uid",
			Controller:         pointer.To(false),
			BlockOwnerDeletion: pointer.To(false),
		},
		{
			APIVersion:         "instana.io/v1",
			Kind:               "InstanaAgent",
			Name:               "instana-agent",
			UID:                "new-uid-value",
			Controller:         pointer.To(true),
			BlockOwnerDeletion: pointer.To(true),
		},
	}

	assertions.Equal(expectedRefs, configMap.OwnerReferences)
}
