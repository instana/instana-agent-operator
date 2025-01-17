/*
(c) Copyright IBM Corp. 2024
*/

package secrets

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	apiV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type CoreV1Mock struct {
	coreV1.CoreV1Interface
	mock.Mock
}

type ConfigMapMock struct {
	coreV1.ConfigMapInterface
	mock.Mock
}

func (mock *CoreV1Mock) ConfigMaps(namespace string) coreV1.ConfigMapInterface {
	args := mock.Called()
	return args.Get(0).(*ConfigMapMock)
}

func (mock *ConfigMapMock) List(ctx context.Context, opts metaV1.ListOptions) (*apiV1.ConfigMapList, error) {
	args := mock.Called()
	return args.Get(0).(*apiV1.ConfigMapList), nil
}

func TestMergeConfigurationYamlWithNoConfigMaps(t *testing.T) {
	client := new(CoreV1Mock)
	configMaps := new(ConfigMapMock)
	client.On("ConfigMaps").Return(configMaps).Once()
	configMapsList := new(apiV1.ConfigMapList)
	configMapsList.Items = []apiV1.ConfigMap{}
	configMaps.On("List").Return(configMapsList).Once()

	merger := NewConfigMergerBuilder(client)
	config_bytes := merger.MergeConfigurationYaml("c: d\n")

	assert.Equal(t, "c: d\n", string(config_bytes))
}

func TestMergeConfigurationYamlWithOtherConfigMapData(t *testing.T) {
	client := new(CoreV1Mock)
	configMaps := new(ConfigMapMock)
	client.On("ConfigMaps").Return(configMaps).Once()
	configMapsList := new(apiV1.ConfigMapList)
	configMapsList.Items = []apiV1.ConfigMap{{Data: map[string]string{"a": "b"}}}
	configMaps.On("List").Return(configMapsList).Once()

	merger := NewConfigMergerBuilder(client)
	config_bytes := merger.MergeConfigurationYaml("c: d\n")

	assert.Equal(t, "c: d\n", string(config_bytes))
}

func TestMergeConfigurationYamlWithEmptyConfigMapData(t *testing.T) {
	client := new(CoreV1Mock)
	configMaps := new(ConfigMapMock)
	client.On("ConfigMaps").Return(configMaps).Once()
	configMapsList := new(apiV1.ConfigMapList)
	configMapsList.Items = []apiV1.ConfigMap{{Data: map[string]string{"configuration_yaml": ""}}}
	configMaps.On("List").Return(configMapsList).Once()

	merger := NewConfigMergerBuilder(client)
	config_bytes := merger.MergeConfigurationYaml("c: d\n")

	assert.Equal(t, "c: d\n", string(config_bytes))
}

func TestMergeConfigurationYamlWithNewTopLevelKey(t *testing.T) {
	client := new(CoreV1Mock)
	configMaps := new(ConfigMapMock)
	client.On("ConfigMaps").Return(configMaps).Once()
	configMapsList := new(apiV1.ConfigMapList)
	configMapsList.Items = []apiV1.ConfigMap{{Data: map[string]string{"configuration_yaml": "a:\n b:\n  - 1\n"}}}
	configMaps.On("List").Return(configMapsList).Once()

	merger := NewConfigMergerBuilder(client)
	config_bytes := merger.MergeConfigurationYaml("c: d\n")

	assert.Equal(t, "a:\n    b:\n        - 1\nc: d\n", string(config_bytes))
}

func TestMergeConfigurationYamlWithNewListItem(t *testing.T) {
	client := new(CoreV1Mock)
	configMaps := new(ConfigMapMock)
	client.On("ConfigMaps").Return(configMaps).Once()
	configMapsList := new(apiV1.ConfigMapList)
	configMapsList.Items = []apiV1.ConfigMap{{Data: map[string]string{"configuration_yaml": "a:\n b:\n  - 2\n"}}}
	configMaps.On("List").Return(configMapsList).Once()

	merger := NewConfigMergerBuilder(client)
	config_bytes := merger.MergeConfigurationYaml("a:\n b:\n  - 1\nc: d\n")

	assert.Equal(t, "a:\n    b:\n        - 1\n        - 2\nc: d\n", string(config_bytes))
}

func TestMergeConfigurationYamlWithMultipleNewListItems(t *testing.T) {
	client := new(CoreV1Mock)
	configMaps := new(ConfigMapMock)
	client.On("ConfigMaps").Return(configMaps).Once()
	configMapsList := new(apiV1.ConfigMapList)
	configMapsList.Items = []apiV1.ConfigMap{{Data: map[string]string{"configuration_yaml": "a:\n b:\n  - 2\n  - 3\n"}}}
	configMaps.On("List").Return(configMapsList).Once()

	merger := NewConfigMergerBuilder(client)
	config_bytes := merger.MergeConfigurationYaml("a:\n b:\n  - 1\nc: d\n")

	assert.Equal(t, "a:\n    b:\n        - 1\n        - 2\n        - 3\nc: d\n", string(config_bytes))
}

func TestMergeConfigurationYamlForFailedRetrieval(t *testing.T) {
	client := new(CoreV1Mock)
	configMaps := new(ConfigMapMock)
	client.On("ConfigMaps").Return(configMaps).Once()
	configMaps.On("List").Return(&apiV1.ConfigMapList{}, errors.New("Failed")).Once()

	merger := NewConfigMergerBuilder(client)
	config_bytes := merger.MergeConfigurationYaml("a:\n b:\n  - 1\nc: d\n")

	assert.Equal(t, "a:\n    b:\n        - 1\nc: d\n", string(config_bytes))
}
