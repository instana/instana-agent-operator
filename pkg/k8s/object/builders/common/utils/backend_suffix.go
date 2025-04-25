/*
(c) Copyright IBM Corp. 2025
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

package utils

import (
	"crypto/sha256"
	"fmt"
)

func GetBackendHash(suffixInput string) string {
	h := sha256.New()
	h.Write([]byte(suffixInput))
	// keep 10 characters of the sha sum and the dash
	// this should sufficiently be unique (like git short commits) and is working fine within naming length constraints in Kubernetes
	return fmt.Sprintf("-%x", h.Sum(nil))[:11]
}
