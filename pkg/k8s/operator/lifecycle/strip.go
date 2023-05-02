package lifecycle

// TODO: Test

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type objectStrip interface {
	stripObject(original client.Object) unstructured.Unstructured
}

type strip struct{}

func (s *strip) stripObject(original client.Object) unstructured.Unstructured {
	res := unstructured.Unstructured{}

	res.SetGroupVersionKind(original.GetObjectKind().GroupVersionKind())
	res.SetName(original.GetName())
	res.SetNamespace(original.GetNamespace())

	return res
}
