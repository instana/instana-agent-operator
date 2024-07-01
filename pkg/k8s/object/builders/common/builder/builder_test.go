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

package builder_test

import (
	"testing"

	"github.com/Masterminds/goutils"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/mocks"
	bldr "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/transformations"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

func newDummyObject() optional.Optional[client.Object] {
	return optional.Of[client.Object](&unstructured.Unstructured{})
}

func TestBuilderTransformerApply(t *testing.T) {
	for _, test := range []struct {
		name     string
		expected func(builder *mocks.MockObjectBuilder) optional.Optional[client.Object]
	}{
		{
			name: "empty_object",
			expected: func(
				builder *mocks.MockObjectBuilder,
			) optional.Optional[client.Object] {
				builder.EXPECT().Build().Return(optional.Empty[client.Object]())

				return optional.Empty[client.Object]()
			},
		},
		{
			name: "non_namespaced",
			expected: func(
				builder *mocks.MockObjectBuilder,
			) optional.Optional[client.Object] {
				componentName, _ := goutils.RandomAlphabetic(10)

				builder.EXPECT().Build().Return(newDummyObject())
				builder.EXPECT().ComponentName().Return(componentName)
				builder.EXPECT().IsNamespaced().Return(false)

				return newDummyObject()
			},
		},
		{
			name: "namespaced",
			expected: func(
				builder *mocks.MockObjectBuilder,
			) optional.Optional[client.Object] {
				componentName, _ := goutils.RandomAlphabetic(10)

				builder.EXPECT().Build().Return(newDummyObject())
				builder.EXPECT().ComponentName().Return(componentName)
				builder.EXPECT().IsNamespaced().Return(true)

				return newDummyObject()
			},
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)
				ctrl := gomock.NewController(t)

				builder := mocks.NewMockObjectBuilder(ctrl)
				transformations := transformations.NewTransformations(&instanav1.InstanaAgent{})

				expected := test.expected(builder).Get()

				bt := bldr.NewBuilderTransformer(transformations)

				actual := bt.Apply(builder).Get()
				if expected != nil {
					assertions.Equal(expected.GetName(), actual.GetName())
				} else {
					assertions.Nil(actual)
				}
			},
		)
	}
}
