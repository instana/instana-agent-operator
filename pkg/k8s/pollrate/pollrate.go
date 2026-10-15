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
// ignored, and reported in Resolution.Warning when it mentions the kubernetes plugin.
func Resolve(configurationYaml string, crPollRate string) Resolution {
	if !crPollRatePattern.MatchString(crPollRate) {
		crPollRate = ""
	}

	fromConfigurationYaml, err := fromConfigurationYaml(configurationYaml)
	if err != nil && !strings.Contains(configurationYaml, PluginKey) {
		// A configuration_yaml that cannot be read only matters here when it could hold the poll rate, so nobody
		// gets a poll rate warning for a setting they never touched.
		err = nil
	}
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
		if crPollRate != "" && !sameSeconds(crPollRate, fromConfigurationYaml) {
			res.Conflict = &Conflict{ConfigurationYaml: fromConfigurationYaml, CR: crPollRate}
		}
		return res
	case crPollRate != "":
		return Resolution{Value: crPollRate, Source: SourceCR, Warning: err}
	default:
		return Resolution{Value: Default, Source: SourceDefault, Warning: err}
	}
}

// sameSeconds compares two "<n>s" values by number, so "010s" and "10s" agree.
func sameSeconds(a string, b string) bool {
	x, errA := strconv.ParseInt(strings.TrimSuffix(a, "s"), 10, 64)
	y, errB := strconv.ParseInt(strings.TrimSuffix(b, "s"), 10, 64)
	return errA == nil && errB == nil && x == y
}

// fromConfigurationYaml returns the poll rate in configurationYaml as "<n>s", or "" when it is not set. It walks the
// YAML nodes instead of decoding into a map because yaml.v3 rejects duplicate keys in a map, while the agent reads
// configuration.yaml with Jackson, which accepts them and keeps the last one.
func fromConfigurationYaml(configurationYaml string) (string, error) {
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(configurationYaml), &document); err != nil {
		return "", errors.Wrap(err, "not valid YAML")
	}
	if len(document.Content) == 0 {
		return "", nil
	}

	plugin, err := lastValue(document.Content[0], PluginKey, "the top level")
	if err != nil || plugin == nil {
		return "", err
	}
	node, err := lastValue(plugin, PollRateKey, PluginKey)
	if err != nil || node == nil {
		return "", err
	}

	var value any
	if err := node.Decode(&value); err != nil {
		return "", errors.Wrapf(err, "%s cannot be read", PollRateKey)
	}
	if value == nil {
		return "", nil
	}
	seconds, err := toSeconds(value)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%ds", seconds), nil
}

// lastValue returns the value of the last occurrence of key in mapping, or nil when mapping is null or does not
// have the key. name describes mapping in the error when it is not a map.
func lastValue(mapping *yaml.Node, key string, name string) (*yaml.Node, error) {
	for mapping.Kind == yaml.AliasNode {
		mapping = mapping.Alias
	}
	if mapping.Kind == yaml.ScalarNode && mapping.ShortTag() == "!!null" {
		return nil, nil
	}
	if mapping.Kind != yaml.MappingNode {
		return nil, errors.Errorf("%s is not a map", name)
	}

	var value *yaml.Node
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if k := mapping.Content[i]; k.Kind == yaml.ScalarNode && k.Value == key {
			value = mapping.Content[i+1]
		}
	}
	return value, nil
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
