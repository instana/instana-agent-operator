/*
(c) Copyright IBM Corp. 2026
*/

// Package pollrate resolves the poll rate the operator passes to k8sensor as -pollrate.
package pollrate

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// Source names where the resolved poll rate came from.
type Source string

const (
	SourceConfigurationYaml Source = "spec.agent.configuration_yaml"
	SourceCR                Source = "spec.k8s_sensor.pollrate"
	SourceDefault           Source = "default"
)

const (
	// PluginKey and PollRateKey locate the poll rate in the agent configuration.yaml.
	PluginKey   = "com.instana.plugin.kubernetes"
	PollRateKey = "poll_rate"

	// Default is passed when nothing is set. k8sensor binaries that predate its own 10s default fall back to 1s,
	// so the operator keeps passing it explicitly.
	Default = "10s"
)

// crPollRatePattern matches the CRD validation of spec.k8s_sensor.pollrate.
var crPollRatePattern = regexp.MustCompile(`^[0-9]+s$`)

// Conflict holds both configured values when they differ. The configuration.yaml value wins.
type Conflict struct {
	ConfigurationYaml string
	CR                string
}

type Resolution struct {
	// Value is passed to k8sensor as -pollrate, e.g. "30s".
	Value  string
	Source Source
	// Conflict is set when both configuration.yaml and the CR field are set and differ.
	Conflict *Conflict
	// Warning is set when the configuration.yaml value could not be used and was ignored.
	Warning error
}

// Resolve picks the k8sensor poll rate: com.instana.plugin.kubernetes.poll_rate in configurationYaml wins, then
// crPollRate (spec.k8s_sensor.pollrate), then Default. It never fails: a configurationYaml that cannot be read is
// ignored and reported in Resolution.Warning.
func Resolve(configurationYaml string, crPollRate string) Resolution {
	if !crPollRatePattern.MatchString(crPollRate) {
		crPollRate = ""
	}

	fromConfigurationYaml, err := fromConfigurationYaml(configurationYaml)
	if err != nil {
		err = errors.Wrapf(
			err,
			"ignoring %s.%s in %s",
			PluginKey,
			PollRateKey,
			SourceConfigurationYaml,
		)
	}

	switch {
	case fromConfigurationYaml != "":
		res := Resolution{Value: fromConfigurationYaml, Source: SourceConfigurationYaml}
		if crPollRate != "" && crPollRate != fromConfigurationYaml {
			res.Conflict = &Conflict{ConfigurationYaml: fromConfigurationYaml, CR: crPollRate}
		}
		return res
	case crPollRate != "":
		return Resolution{Value: crPollRate, Source: SourceCR, Warning: err}
	default:
		return Resolution{Value: Default, Source: SourceDefault, Warning: err}
	}
}

// fromConfigurationYaml returns the poll rate in configurationYaml as "<n>s", or "" when it is not set.
func fromConfigurationYaml(configurationYaml string) (string, error) {
	var config map[string]any
	if err := yaml.Unmarshal([]byte(configurationYaml), &config); err != nil {
		return "", errors.Wrap(err, "not valid YAML")
	}

	plugin, ok := config[PluginKey]
	if !ok || plugin == nil {
		return "", nil
	}
	pluginConfig, ok := plugin.(map[string]any)
	if !ok {
		return "", errors.Errorf("%s is not a map", PluginKey)
	}

	value, ok := pluginConfig[PollRateKey]
	if !ok || value == nil {
		return "", nil
	}
	seconds, err := toSeconds(value)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%ds", seconds), nil
}

// toSeconds reads a poll rate the way the agent does (PollRateUtil): numbers and numeric strings are accepted,
// fractions are truncated, values must be positive and are capped at math.MaxInt32. Snapping to the allowed set
// is left to k8sensor.
func toSeconds(value any) (int64, error) {
	var f float64
	switch v := value.(type) {
	case int:
		f = float64(v)
	case uint64:
		f = float64(v)
	case float64:
		f = v
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return 0, errors.Errorf("%s %q is not a number", PollRateKey, v)
		}
		f = parsed
	default:
		return 0, errors.Errorf("%s %v is not a number", PollRateKey, v)
	}

	seconds := math.Trunc(f)
	if math.IsNaN(seconds) || seconds <= 0 {
		return 0, errors.Errorf("%s %v must be positive", PollRateKey, value)
	}
	return int64(math.Min(seconds, math.MaxInt32)), nil
}
