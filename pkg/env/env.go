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

package env

import (
	"os"
)

var OPERATOR_VERSION_ENVVAR = "OPERATOR_VERSION"
var FALLBACK_OPERATOR_VERSION = "v0.0.1-dev"

func GetOperatorVersion() string {
	if os.Getenv(OPERATOR_VERSION_ENVVAR) != "" {
		return os.Getenv(OPERATOR_VERSION_ENVVAR)
	}
	return FALLBACK_OPERATOR_VERSION
}
