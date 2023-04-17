package v1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/instana/instana-agent-operator/pkg/or_die"
	"github.com/instana/instana-agent-operator/pkg/pointer"
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

var (
	allPossibleEnabled = []Enabled{
		{},
		{
			Enabled: pointer.To(true),
		},
		{
			Enabled: pointer.To(false),
		},
	}
	allPossibleEnabledPtr = []*Enabled{
		nil,
		{},
		{
			Enabled: pointer.To(true),
		},
		{
			Enabled: pointer.To(false),
		},
	}
)

func jsonStringOrDie(obj interface{}) string {
	return string(
		or_die.New[[]byte]().ResultOrDie(
			func() ([]byte, error) {
				return json.Marshal(obj)
			},
		),
	)
}

func testForOtlp(t *testing.T, otlp *OpenTelemetry, getExpected func(otlp *OpenTelemetry) bool, getActual func() bool) {
	t.Run(
		jsonStringOrDie(otlp), func(t *testing.T) {
			assertions := require.New(t)

			expected := getExpected(otlp)
			actual := getActual()

			assertions.Equal(expected, actual)
		},
	)
}

func grpcIsEnabled_expected(otlp *OpenTelemetry) bool {
	switch grpc := otlp.GRPC; grpc {
	case nil:
		switch enabled := otlp.Enabled.Enabled; enabled {
		case nil:
			return false
		default:
			return *enabled
		}
	default:
		switch enabled := grpc.Enabled; enabled {
		case nil:
			return true
		default:
			return *enabled
		}
	}
}

func TestOpenTelemetry_GrpcIsEnabled(t *testing.T) {
	for _, enabled := range allPossibleEnabled {
		for _, grpc := range allPossibleEnabledPtr {
			otlp := &OpenTelemetry{
				Enabled: enabled,
				GRPC:    grpc,
			}
			testForOtlp(t, otlp, grpcIsEnabled_expected, otlp.GrpcIsEnabled)
		}
	}
}

func httpIsEnabled_expected(otlp *OpenTelemetry) bool {
	switch http := otlp.HTTP; http {
	case nil:
		return false
	default:
		switch enabled := http.Enabled; enabled {
		case nil:
			return true
		default:
			return *enabled
		}
	}
}

func TestOpenTelemetry_HttpIsEnabled(t *testing.T) {
	for _, http := range allPossibleEnabledPtr {
		otlp := &OpenTelemetry{
			HTTP: http,
		}
		testForOtlp(t, otlp, httpIsEnabled_expected, otlp.HttpIsEnabled)
	}
}

func isEnabled_expected(otlp *OpenTelemetry) bool {
	return grpcIsEnabled_expected(otlp) || httpIsEnabled_expected(otlp)
}

func TestOpenTelemetry_IsEnabled(t *testing.T) {
	for _, enabled := range allPossibleEnabled {
		for _, grpc := range allPossibleEnabledPtr {
			for _, http := range allPossibleEnabledPtr {
				otlp := &OpenTelemetry{
					Enabled: enabled,
					GRPC:    grpc,
					HTTP:    http,
				}
				testForOtlp(t, otlp, isEnabled_expected, otlp.IsEnabled)
			}
		}
	}
}
