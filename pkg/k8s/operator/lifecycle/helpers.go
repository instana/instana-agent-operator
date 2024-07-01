/*
(c) Copyright IBM Corp. 2024
(c) Copyright Instana Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package lifecycle

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// asUnstructured is a simple utility function to convert controller client.Object to unstructured.Unstructured
func asUnstructured(obj client.Object) unstructured.Unstructured {
	res := unstructured.Unstructured{}

	res.SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())
	res.SetName(obj.GetName())
	res.SetNamespace(obj.GetNamespace())

	return res
}

// asUnstructureds is a simple utility function to convert a list of client.Object to a list of unstructured.Unstructured
func asUnstructureds(objects ...client.Object) []unstructured.Unstructured {
	var unstructureds []unstructured.Unstructured
	for _, obj := range objects {
		unstructureds = append(unstructureds, asUnstructured(obj))
	}
	return unstructureds
}

// asObject is a simple utility function to convert unstructured.Unstructured to controller client.Object
func asObject(val unstructured.Unstructured) client.Object {
	return &val
}
