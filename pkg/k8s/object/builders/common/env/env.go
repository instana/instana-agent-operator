// /*
// (c) Copyright IBM Corp. 2024
// (c) Copyright Instana Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */
package env

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/instana/instana-agent-operator/pkg/optional"
)

func fromCRField[T any](name string, val T) optional.Optional[corev1.EnvVar] {
	return optional.Map(
		optional.Of(val), func(v T) corev1.EnvVar {
			return corev1.EnvVar{
				Name:  name,
				Value: fmt.Sprintf("%v", v),
			}
		},
	)
}
