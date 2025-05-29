/*
(c) Copyright IBM Corp. 2025

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

package serviceaccount

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/mocks"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

func TestRemoteServiceAccountBuilder_IsNamespaced_ComponentName(t *testing.T) {
	assertions := require.New(t)

	sb := NewServiceAccountBuilder(nil)

	assertions.True(sb.IsNamespaced())
	assertions.Equal(constants.ComponentRemoteAgent, sb.ComponentName())
}

func TestRemoteServiceAccountBuilder_Build(t *testing.T) {
	for _, test := range []struct {
		createServiceAccount *bool
	}{
		{
			createServiceAccount: pointer.To(true),
		},
		{
			createServiceAccount: pointer.To(false),
		},
		{
			createServiceAccount: nil,
		},
	} {
		t.Run(
			fmt.Sprintf("%+v", test), func(t *testing.T) {
				assertions := require.New(t)
				ctrl := gomock.NewController(t)

				serviceAccountName := "remote-agent"
				namespace := rand.String(10)

				const annotationKeyName = "instana.io/example"
				annotationKeyValue := rand.String(10)

				agent := &instanav1.RemoteAgent{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespace,
					},
					Spec: instanav1.RemoteAgentSpec{
						ServiceAccountSpec: instanav1.ServiceAccountSpec{
							Create: instanav1.Create{
								Create: test.createServiceAccount,
							},
							Annotations: map[string]string{
								annotationKeyName: annotationKeyValue,
							},
						},
					},
				}

				expected := optional.Of[client.Object](
					&corev1.ServiceAccount{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "v1",
							Kind:       "ServiceAccount",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      serviceAccountName,
							Namespace: namespace,
							Annotations: map[string]string{
								annotationKeyName: annotationKeyValue,
							},
						},
					},
				)

				helpers := mocks.NewMockHelpers(ctrl)

				sb := &serviceAccountBuilder{
					RemoteAgent:   agent,
					RemoteHelpers: helpers,
				}

				actual := sb.Build()

				if pointer.DerefOrEmpty(test.createServiceAccount) {
					assertions.Equal(expected, actual)
				} else {
					assertions.Empty(actual)
				}
			},
		)
	}
}
