package configmap

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

func TestConfigMapBuilder_IsNamespaced_ComponentName(t *testing.T) {
	assertions := require.New(t)

	cmb := NewConfigMapBuilder(nil)

	assertions.True(cmb.IsNamespaced())
	assertions.Equal(constants.ComponentK8Sensor, cmb.ComponentName())
}

func TestConfigMapBuilder_Build(t *testing.T) {
	assertions := require.New(t)
	ctrl := gomock.NewController(t)

	sensorResourcesName := rand.String(10)
	namespace := rand.String(10)

	endpointHost := rand.String(10)
	endpointPort := rand.String(10)

	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				EndpointHost: endpointHost,
				EndpointPort: endpointPort,
			},
		},
	}

	expected := optional.Of[client.Object](
		&corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      sensorResourcesName,
				Namespace: namespace,
			},
			Data: map[string]string{
				"backend": fmt.Sprintf("%s:%s", endpointHost, endpointPort),
			},
		},
	)

	helpers := NewMockHelpers(ctrl)
	helpers.EXPECT().K8sSensorResourcesName().Return(sensorResourcesName)

	cmb := &configMapBuilder{
		InstanaAgent: agent,
		Helpers:      helpers,
	}

	actual := cmb.Build()

	assertions.Equal(expected, actual)
}
