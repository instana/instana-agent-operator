package v1

import (
	"testing"

	"github.com/stretchr/testify/require"
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
