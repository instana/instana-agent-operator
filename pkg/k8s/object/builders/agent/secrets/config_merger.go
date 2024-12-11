/*
(c) Copyright IBM Corp. 2024
*/

package secrets

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	"gopkg.in/yaml.v3"
	apiV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const ConfigMapLabel = "instana.io/agent-config=true"

type ConfigMerger struct {
	logger    logr.Logger
	k8sClient v1.CoreV1Interface
}

func NewConfigMergerBuilder(client v1.CoreV1Interface) *ConfigMerger {
	return &ConfigMerger{
		logger:    logf.Log.WithName("instana-agent-config-merger"),
		k8sClient: client,
	}
}

func (c *ConfigMerger) MergeConfigurationYaml(agentConfiguration string) []byte {
	operator_data := make(map[string]interface{})
	err := yaml.Unmarshal([]byte([]byte(agentConfiguration)), operator_data)
	config := []byte{}
	if err != nil {
		c.logger.Error(err, "Failed to load agent configuration")
	} else {
		config_maps := c.fetchConfigMaps()
		for _, config_map := range config_maps {
			config_map_data := make(map[string]interface{})
			err := yaml.Unmarshal([]byte(config_map.Data["configuration_yaml"]), &config_map_data)
			if err != nil {
				c.logger.Error(err, "Failed to parse agent configuration YAML")
			} else {
				operator_data = c.mergeConfig(operator_data, config_map_data)
			}
		}
		config, err = yaml.Marshal(operator_data)
	}
	return config
}

func (c *ConfigMerger) mergeConfig(operator_data, config_map_data map[string]interface{}) map[string]interface{} {
	for key, cm_value := range config_map_data {
		if op_value, ok := operator_data[key]; ok {
			op_value_kind := reflect.TypeOf(op_value).Kind()
			if op_value_kind == reflect.Array || op_value_kind == reflect.Slice {
				operator_data[key] = append(op_value.([]interface{}), cm_value.([]interface{})...)
			} else {
				c.mergeConfig(operator_data[key].(map[string]interface{}), cm_value.(map[string]interface{}))
			}
		} else {
			operator_data[key] = cm_value
		}
	}
	return operator_data
}

func (c *ConfigMerger) fetchConfigMaps() []apiV1.ConfigMap {
	config_maps := []apiV1.ConfigMap{}
	c.logger.Info(fmt.Sprintf("Fetching agent configmaps with label '%s'", ConfigMapLabel))
	config_map_list, err := c.k8sClient.ConfigMaps("").List(context.TODO(), metav1.ListOptions{LabelSelector: ConfigMapLabel})
	if err != nil {
		c.logger.Error(err, fmt.Sprintf("Failed to fetch agent configmaps with label '%s'", ConfigMapLabel))
	} else {
		config_maps = config_map_list.Items
		c.logger.Info(fmt.Sprintf("Found %d configmaps with label '%s'", len(config_maps), ConfigMapLabel))
	}
	return config_maps
}
