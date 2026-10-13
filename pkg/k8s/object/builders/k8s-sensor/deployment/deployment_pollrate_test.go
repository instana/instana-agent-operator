/*
(c) Copyright IBM Corp. 2026
*/

package deployment

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// builderForPollRateTests returns a builder for an agent defaulted the same way the controller defaults it before the
// builders run, so a default applied in Default() cannot hide an unset pollrate from the resolver.
func builderForPollRateTests(configurationYaml string, crPollRate string) *deploymentBuilder {
	agent := agentForETCDEnvTests()
	agent.Spec.Agent.ConfigurationYaml = configurationYaml
	agent.Spec.K8sSensor.PollRate = crPollRate
	agent.Default()
	return builderFor(agent, false, &DeploymentContext{})
}

func TestDeploymentBuilder_PollRateArg(t *testing.T) {
	for _, test := range []struct {
		name              string
		configurationYaml string
		crPollRate        string
		expected          string
	}{
		{
			name:     "neither set passes 10s",
			expected: "10s",
		},
		{
			name:       "only CR set",
			crPollRate: "20s",
			expected:   "20s",
		},
		{
			name:              "only configuration_yaml set",
			configurationYaml: "com.instana.plugin.kubernetes:\n  poll_rate: 30\n",
			expected:          "30s",
		},
		{
			name:              "configuration_yaml wins over the CR",
			configurationYaml: "com.instana.plugin.kubernetes:\n  poll_rate: 60\n",
			crPollRate:        "10s",
			expected:          "60s",
		},
		{
			name:              "invalid configuration_yaml falls back to the CR",
			configurationYaml: "com.instana.plugin.kubernetes:\n  poll_rate: [60\n",
			crPollRate:        "20s",
			expected:          "20s",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			args := builderForPollRateTests(
				test.configurationYaml,
				test.crPollRate,
			).getK8SensorArgs()

			i := indexOf(args, "-pollrate")
			require.GreaterOrEqual(t, i, 0, "-pollrate should always be passed")
			require.Less(t, i+1, len(args))
			assert.Equal(t, test.expected, args[i+1])
		})
	}
}

// TestDeploymentBuilder_UnrelatedConfigurationYamlEditKeepsDeployment covers that only the resolved poll rate
// reaches the Deployment. Anything else in configuration_yaml reaches k8sensor through the mounted config Secret,
// so editing it must not change the pod template and roll every k8sensor replica.
func TestDeploymentBuilder_UnrelatedConfigurationYamlEditKeepsDeployment(t *testing.T) {
	before := builderForPollRateTests(
		"com.instana.plugin.kubernetes:\n  poll_rate: 30\ncom.instana.plugin.host:\n  tags: [a]\n",
		"10s",
	).build()

	t.Run("unrelated keys changed", func(t *testing.T) {
		after := builderForPollRateTests(
			"com.instana.plugin.kubernetes:\n  poll_rate: 30\n  enabled: true\n"+
				"com.instana.plugin.host:\n  tags: [a, b]\n",
			"10s",
		).build()

		assert.Equal(t, before.Spec, after.Spec)
	})

	t.Run("resolved poll rate changed", func(t *testing.T) {
		after := builderForPollRateTests(
			"com.instana.plugin.kubernetes:\n  poll_rate: 60\ncom.instana.plugin.host:\n  tags: [a]\n",
			"10s",
		).build()

		assert.NotEqual(t, before.Spec, after.Spec)
	})
}
