/*
(c) Copyright IBM Corp. 2026
*/

package pollrate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolve(t *testing.T) {
	for _, test := range []struct {
		name              string
		configurationYaml string
		crPollRate        string
		expected          Resolution
		expectWarning     string
	}{
		{
			name:     "neither set passes the default",
			expected: Resolution{Value: "10s", Source: SourceDefault},
		},
		{
			name:              "only configuration_yaml set",
			configurationYaml: "com.instana.plugin.kubernetes:\n  poll_rate: 30\n",
			expected:          Resolution{Value: "30s", Source: SourceConfigurationYaml},
		},
		{
			name:       "only CR set",
			crPollRate: "20s",
			expected:   Resolution{Value: "20s", Source: SourceCR},
		},
		{
			name:              "both set and equal",
			configurationYaml: "com.instana.plugin.kubernetes:\n  poll_rate: 20\n",
			crPollRate:        "20s",
			expected:          Resolution{Value: "20s", Source: SourceConfigurationYaml},
		},
		{
			name:              "both set and different, configuration_yaml wins",
			configurationYaml: "com.instana.plugin.kubernetes:\n  poll_rate: 60\n",
			crPollRate:        "10s",
			expected: Resolution{
				Value:    "60s",
				Source:   SourceConfigurationYaml,
				Conflict: &Conflict{ConfigurationYaml: "60s", CR: "10s"},
			},
		},
		{
			name:              "numeric string value",
			configurationYaml: "com.instana.plugin.kubernetes:\n  poll_rate: \" 30 \"\n",
			expected:          Resolution{Value: "30s", Source: SourceConfigurationYaml},
		},
		{
			name:              "float value is truncated like the agent does",
			configurationYaml: "com.instana.plugin.kubernetes:\n  poll_rate: 5.9\n",
			expected:          Resolution{Value: "5s", Source: SourceConfigurationYaml},
		},
		{
			name:              "float string value is truncated like the agent does",
			configurationYaml: "com.instana.plugin.kubernetes:\n  poll_rate: \"20.5\"\n",
			expected:          Resolution{Value: "20s", Source: SourceConfigurationYaml},
		},
		{
			name:              "huge value is capped like the agent does",
			configurationYaml: "com.instana.plugin.kubernetes:\n  poll_rate: 99999999999999999999\n",
			expected:          Resolution{Value: "2147483647s", Source: SourceConfigurationYaml},
		},
		{
			name:              "other keys under the kubernetes plugin are ignored",
			configurationYaml: "com.instana.plugin.kubernetes:\n  enabled: false\n",
			crPollRate:        "5s",
			expected:          Resolution{Value: "5s", Source: SourceCR},
		},
		{
			name:              "other plugins are ignored",
			configurationYaml: "com.instana.plugin.host:\n  poll_rate: 30\n",
			expected:          Resolution{Value: "10s", Source: SourceDefault},
		},
		{
			name:              "empty kubernetes plugin section is ignored",
			configurationYaml: "com.instana.plugin.kubernetes:\n",
			expected:          Resolution{Value: "10s", Source: SourceDefault},
		},
		{
			name:              "null poll_rate is ignored",
			configurationYaml: "com.instana.plugin.kubernetes:\n  poll_rate: null\n",
			crPollRate:        "5s",
			expected:          Resolution{Value: "5s", Source: SourceCR},
		},
		{
			name:              "comment only configuration_yaml is ignored",
			configurationYaml: "# nothing configured yet\n",
			expected:          Resolution{Value: "10s", Source: SourceDefault},
		},
		{
			name:              "invalid YAML falls back to the CR and warns",
			configurationYaml: "com.instana.plugin.kubernetes:\n  poll_rate: [30\n",
			crPollRate:        "20s",
			expected:          Resolution{Value: "20s", Source: SourceCR},
			expectWarning:     "not valid YAML",
		},
		{
			name:              "invalid YAML without a CR value falls back to the default and warns",
			configurationYaml: "\tnot: yaml",
			expected:          Resolution{Value: "10s", Source: SourceDefault},
			expectWarning:     "not valid YAML",
		},
		{
			name:              "configuration_yaml that is not a map warns",
			configurationYaml: "- com.instana.plugin.kubernetes\n",
			crPollRate:        "20s",
			expected:          Resolution{Value: "20s", Source: SourceCR},
			expectWarning:     "not valid YAML",
		},
		{
			name:              "kubernetes plugin section that is not a map warns",
			configurationYaml: "com.instana.plugin.kubernetes: 30\n",
			crPollRate:        "20s",
			expected:          Resolution{Value: "20s", Source: SourceCR},
			expectWarning:     "is not a map",
		},
		{
			name:              "non numeric value falls back to the CR and warns",
			configurationYaml: "com.instana.plugin.kubernetes:\n  poll_rate: 30s\n",
			crPollRate:        "20s",
			expected:          Resolution{Value: "20s", Source: SourceCR},
			expectWarning:     "is not a number",
		},
		{
			name:              "boolean value falls back to the CR and warns",
			configurationYaml: "com.instana.plugin.kubernetes:\n  poll_rate: true\n",
			crPollRate:        "20s",
			expected:          Resolution{Value: "20s", Source: SourceCR},
			expectWarning:     "is not a number",
		},
		{
			name:              "zero falls back to the CR and warns",
			configurationYaml: "com.instana.plugin.kubernetes:\n  poll_rate: 0\n",
			crPollRate:        "20s",
			expected:          Resolution{Value: "20s", Source: SourceCR},
			expectWarning:     "must be positive",
		},
		{
			name:              "negative value falls back to the CR and warns",
			configurationYaml: "com.instana.plugin.kubernetes:\n  poll_rate: -5\n",
			crPollRate:        "20s",
			expected:          Resolution{Value: "20s", Source: SourceCR},
			expectWarning:     "must be positive",
		},
		{
			name:              "NaN falls back to the CR and warns",
			configurationYaml: "com.instana.plugin.kubernetes:\n  poll_rate: .nan\n",
			crPollRate:        "20s",
			expected:          Resolution{Value: "20s", Source: SourceCR},
			expectWarning:     "must be positive",
		},
		{
			name:       "CR value not matching the CRD pattern is treated as unset",
			crPollRate: "1m",
			expected:   Resolution{Value: "10s", Source: SourceDefault},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			actual := Resolve(test.configurationYaml, test.crPollRate)

			if test.expectWarning == "" {
				require.NoError(t, actual.Warning)
			} else {
				require.Error(t, actual.Warning)
				assert.Contains(t, actual.Warning.Error(), test.expectWarning)
			}
			actual.Warning = nil
			assert.Equal(t, test.expected, actual)
		})
	}
}
