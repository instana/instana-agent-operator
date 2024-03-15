package lifecycle

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_strip_stripObject(t *testing.T) {
	assertions := require.New(t)

	original := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cm-name",
			Namespace: "cm-ns",
		},
	}

	actual := (&strip{}).stripObject(original)

	assertions.Equal("cm-name", actual.GetName())
	assertions.Equal("cm-ns", actual.GetNamespace())
	assertions.Equal("v1", actual.GetAPIVersion())
	assertions.Equal("ConfigMap", actual.GetKind())
}
