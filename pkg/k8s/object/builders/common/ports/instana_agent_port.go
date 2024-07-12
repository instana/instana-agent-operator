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

package ports

import (
	"errors"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
)

type InstanaAgentPort string

const (
	AgentAPIsPort                 InstanaAgentPort = "agent-apis"
	AgentAPIsPortNumber           int32            = 42699
	AgentSocketPort               InstanaAgentPort = "agent-socket"
	AgentSocketPortNumber         int32            = 42666
	OpenTelemetryLegacyPort       InstanaAgentPort = "otlp-legacy"
	OpenTelemetryLegacyPortNumber int32            = 55680
	OpenTelemetryGRPCPort         InstanaAgentPort = "otlp-grpc"
	OpenTelemetryGRPCPortNumber   int32            = 4317
	OpenTelemetryHTTPPort         InstanaAgentPort = "otlp-http"
	OpenTelemetryHTTPPortNumber   int32            = 4318
)

func (p InstanaAgentPort) String() string {
	return string(p)
}

func (p InstanaAgentPort) PortNumber() int32 {
	switch p {
	case AgentAPIsPort:
		return AgentAPIsPortNumber
	case AgentSocketPort:
		return AgentSocketPortNumber
	case OpenTelemetryLegacyPort:
		return OpenTelemetryLegacyPortNumber
	case OpenTelemetryGRPCPort:
		return OpenTelemetryGRPCPortNumber
	case OpenTelemetryHTTPPort:
		return OpenTelemetryHTTPPortNumber
	default:
		panic(errors.New("unknown port requested"))
	}
}

func (p InstanaAgentPort) IsEnabled(openTelemetrySettings instanav1.OpenTelemetry) bool {
	switch p {
	case OpenTelemetryLegacyPort:
		fallthrough
	case OpenTelemetryGRPCPort:
		return openTelemetrySettings.GrpcIsEnabled()
	case OpenTelemetryHTTPPort:
		return openTelemetrySettings.HttpIsEnabled()
	case AgentAPIsPort:
		fallthrough
	case AgentSocketPort:
		fallthrough
	default:
		return true
	}
}
