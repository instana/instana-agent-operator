package v1

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestImageSpec_Image(t *testing.T) {
	t.Run("with digest", func(t *testing.T) {
		assertions := require.New(t)

		i := ImageSpec{
			Name:   "icr.io/instana/instana-agent-operator",
			Digest: "sha256:61417f330b2eb7eff88ccb9812921b65a31bf350fe9efdcb6663a29759c47fe4",
		}

		assertions.Equal("icr.io/instana/instana-agent-operator@sha256:61417f330b2eb7eff88ccb9812921b65a31bf350fe9efdcb6663a29759c47fe4", i.Image())
	})

	t.Run("with digest and tag", func(t *testing.T) {
		assertions := require.New(t)

		i := ImageSpec{
			Name:   "icr.io/instana/instana-agent-operator",
			Digest: "sha256:61417f330b2eb7eff88ccb9812921b65a31bf350fe9efdcb6663a29759c47fe4",
			Tag:    "2.0.10",
		}

		assertions.Equal("icr.io/instana/instana-agent-operator@sha256:61417f330b2eb7eff88ccb9812921b65a31bf350fe9efdcb6663a29759c47fe4", i.Image())
	})

	t.Run("with tag", func(t *testing.T) {
		assertions := require.New(t)

		i := ImageSpec{
			Name: "icr.io/instana/instana-agent-operator",
			Tag:  "2.0.10",
		}

		assertions.Equal("icr.io/instana/instana-agent-operator:2.0.10", i.Image())
	})

	t.Run("with name only", func(t *testing.T) {
		assertions := require.New(t)

		i := ImageSpec{
			Name: "icr.io/instana/instana-agent-operator:2.0.10",
		}

		assertions.Equal("icr.io/instana/instana-agent-operator:2.0.10", i.Image())
	})
}
