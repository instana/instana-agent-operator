/*
(c) Copyright IBM Corp. 2025
(c) Copyright Instana Inc.
*/

package configmap

import (
	"testing"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/stretchr/testify/require"
)

func TestServiceCaConfigMapBuilderIsNamespacedComponentName(t *testing.T) {
	assertions := require.New(t)

	cmb := NewServiceCaConfigMapBuilder(nil)

	assertions.True(cmb.IsNamespaced())
	assertions.Equal(constants.ComponentK8Sensor, cmb.ComponentName())
}

func TestServiceCaConfigMapBuilderBuild(t *testing.T) {
	assertions := require.New(t)

	namespace := "test-namespace"

	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
	}

	cmb := &serviceCaConfigMapBuilder{
		InstanaAgent: agent,
		Helpers:      helpers.NewHelpers(agent),
	}

	actual := cmb.Build()

	assertions.True(actual.IsPresent())
	configMap, ok := actual.Get().(*corev1.ConfigMap)
	assertions.True(ok)
	assertions.Equal("sensor-service-ca", configMap.Name)
	assertions.Equal(namespace, configMap.Namespace)
	assertions.Equal("true", configMap.Annotations["service.beta.openshift.io/inject-cabundle"])
}
