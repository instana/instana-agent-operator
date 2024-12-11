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

type ConfigMerger interface {
	MergeConfigurationYaml(string) []byte
}

type DefaultConfigMerger struct {
	logger    logr.Logger
	k8sClient v1.CoreV1Interface
}

func NewConfigMergerBuilder(client v1.CoreV1Interface) DefaultConfigMerger {
	return DefaultConfigMerger{
		logger:    logf.Log.WithName("instana-agent-config-merger"),
		k8sClient: client,
	}
}

func (c DefaultConfigMerger) MergeConfigurationYaml(agentConfiguration string) []byte {
	agentData := make(map[string]interface{})
	err := yaml.Unmarshal([]byte([]byte(agentConfiguration)), agentData)
	config := []byte{}
	if err != nil {
		c.logger.Error(err, "Failed to load agent configuration")
	} else {
		configMaps := c.fetchConfigMaps()
		for _, configMap := range configMaps {
			configMapData := make(map[string]interface{})
			err := yaml.Unmarshal([]byte(configMap.Data["configuration_yaml"]), &configMapData)
			if err != nil {
				c.logger.Error(err, "Failed to parse agent configuration YAML")
			} else {
				agentData = c.mergeConfig(agentData, configMapData)
			}
		}
		config, err = yaml.Marshal(agentData)
	}
	return config
}

func (c DefaultConfigMerger) mergeConfig(agentData, configMapData map[string]interface{}) map[string]interface{} {
	for key, configMapValue := range configMapData {
		if agentValue, ok := agentData[key]; ok {
			agentValueKind := reflect.TypeOf(agentValue).Kind()
			if agentValueKind == reflect.Array || agentValueKind == reflect.Slice {
				agentData[key] = append(agentValue.([]interface{}), configMapValue.([]interface{})...)
			} else {
				c.mergeConfig(agentData[key].(map[string]interface{}), configMapValue.(map[string]interface{}))
			}
		} else {
			agentData[key] = configMapValue
		}
	}
	return agentData
}

func (c DefaultConfigMerger) fetchConfigMaps() []apiV1.ConfigMap {
	configMaps := []apiV1.ConfigMap{}
	c.logger.Info(fmt.Sprintf("Fetching agent configmaps with label '%s'", ConfigMapLabel))
	configMapList, err := c.k8sClient.ConfigMaps("").List(context.TODO(), metav1.ListOptions{LabelSelector: ConfigMapLabel})
	if err != nil {
		c.logger.Error(err, fmt.Sprintf("Failed to fetch agent configmaps with label '%s'", ConfigMapLabel))
	} else {
		configMaps = configMapList.Items
		c.logger.Info(fmt.Sprintf("Found %d configmaps with label '%s'", len(configMaps), ConfigMapLabel))
	}
	return configMaps
}
