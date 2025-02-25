/*
(c) Copyright IBM Corp. 2024,2025
*/

package v1

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/instana/instana-agent-operator/pkg/map_defaulter"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

func TestImageSpec_Image(t *testing.T) {
	for _, test := range []struct {
		name string
		ImageSpec
		expected string
	}{
		{
			name: "with_digest",
			ImageSpec: ImageSpec{
				Name:   "icr.io/instana/instana-agent-operator",
				Digest: "sha256:61417f330b2eb7eff88ccb9812921b65a31bf350fe9efdcb6663a29759c47fe4",
			},
			expected: "icr.io/instana/instana-agent-operator@sha256:61417f330b2eb7eff88ccb9812921b65a31bf350fe9efdcb6663a29759c47fe4",
		},
		{
			name: "with_digest_and_tag",
			ImageSpec: ImageSpec{
				Name:   "icr.io/instana/instana-agent-operator",
				Digest: "sha256:61417f330b2eb7eff88ccb9812921b65a31bf350fe9efdcb6663a29759c47fe4",
				Tag:    "2.0.10",
			},
			expected: "icr.io/instana/instana-agent-operator@sha256:61417f330b2eb7eff88ccb9812921b65a31bf350fe9efdcb6663a29759c47fe4",
		},
		{
			name: "with_tag",
			ImageSpec: ImageSpec{
				Name: "icr.io/instana/instana-agent-operator",
				Tag:  "2.0.10",
			},
			expected: "icr.io/instana/instana-agent-operator:2.0.10",
		},
		{
			name: "with_name_only",
			ImageSpec: ImageSpec{
				Name: "icr.io/instana/instana-agent-operator:2.0.10",
			},
			expected: "icr.io/instana/instana-agent-operator:2.0.10",
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)

				actual := test.ImageSpec.Image()

				assertions.Equal(test.expected, actual)
			},
		)
	}
}

func TestDaemonSetBuilder_getResourceRequirements(t *testing.T) {
	metaAssertions := require.New(t)

	type testParams struct {
		providedMemRequest string
		providedCpuRequest string
		providedMemLimit   string
		providedCpuLimit   string

		expectedMemRequest string
		expectedCpuRequest string
		expectedMemLimit   string
		expectedCpuLimit   string
	}

	tests := make([]testParams, 0, 16)
	for _, providedMemRequest := range []string{"", "123Mi"} {
		for _, providedCpuRequest := range []string{"", "1.2"} {
			for _, providedMemLimit := range []string{"", "456Mi"} {
				for _, providedCpuLimit := range []string{"", "4.5"} {
					tests = append(
						tests, testParams{
							expectedMemRequest: optional.Of(providedMemRequest).GetOrDefault("768Mi"),
							expectedCpuRequest: optional.Of(providedCpuRequest).GetOrDefault("0.5"),
							expectedMemLimit:   optional.Of(providedMemLimit).GetOrDefault("768Mi"),
							expectedCpuLimit:   optional.Of(providedCpuLimit).GetOrDefault("1.5"),

							providedMemRequest: providedMemRequest,
							providedCpuRequest: providedCpuRequest,
							providedMemLimit:   providedMemLimit,
							providedCpuLimit:   providedCpuLimit,
						},
					)
				}
			}
		}
	}

	metaAssertions.Len(tests, 16)

	for _, test := range tests {
		t.Run(
			fmt.Sprintf("%+v", test), func(t *testing.T) {
				assertions := require.New(t)

				provided := ResourceRequirements{}

				setIfNotEmpty := func(providedVal string, key corev1.ResourceName, resourceList *corev1.ResourceList) {
					if providedVal != "" {
						map_defaulter.NewMapDefaulter((*map[corev1.ResourceName]resource.Quantity)(resourceList)).SetIfEmpty(
							key,
							resource.MustParse(providedVal),
						)
					}
				}

				setIfNotEmpty(test.providedMemLimit, corev1.ResourceMemory, &provided.Limits)
				setIfNotEmpty(test.providedCpuLimit, corev1.ResourceCPU, &provided.Limits)
				setIfNotEmpty(test.providedMemRequest, corev1.ResourceMemory, &provided.Requests)
				setIfNotEmpty(test.providedCpuRequest, corev1.ResourceCPU, &provided.Requests)

				actual := provided.GetOrDefault()

				assertions.Equal(resource.MustParse(test.expectedMemLimit), actual.Limits[corev1.ResourceMemory])
				assertions.Equal(resource.MustParse(test.expectedCpuLimit), actual.Limits[corev1.ResourceCPU])
				assertions.Equal(resource.MustParse(test.expectedMemRequest), actual.Requests[corev1.ResourceMemory])
				assertions.Equal(resource.MustParse(test.expectedCpuRequest), actual.Requests[corev1.ResourceCPU])
			},
		)
	}
}
