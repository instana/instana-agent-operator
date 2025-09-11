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

package constants

// components
const (
	ComponentInstanaAgentRemote = "instana-agent-remote"
	ComponentInstanaAgent       = "instana-agent"
	ComponentK8Sensor           = "k8sensor"
)

// labels
const (
	LabelAgentMode = "instana/agent-mode"
)

// keys
const (
	AgentKey    = "key"
	DownloadKey = "downloadKey"
	BackendKey  = "backend"
)

// ReaderVerbs are the list RBAC Verbs used for being able to read resources for a specific api group as specified in a PolicyRule, i.e: "get", "list", "watch"
func ReaderVerbs() []string {
	return []string{"get", "list", "watch"}
}

const InstanaNamespacesDetailsFileName = "namespaces.yaml"
const InstanaConfigDirectory = "/opt/instana/agent/etc/instana-config-yml"
const InstanaNamespacesDetailsDirectory = "/opt/instana/agent/etc/namespaces"
const InstanaSecretsDirectory = "/opt/instana/agent/etc/instana/secrets"

// Secret file names
const (
	SecretFileAgentKey                  = "INSTANA_AGENT_KEY"
	SecretFileDownloadKey               = "INSTANA_DOWNLOAD_KEY"
	SecretFileProxyUser                 = "INSTANA_AGENT_PROXY_USER"
	SecretFileProxyPassword             = "INSTANA_AGENT_PROXY_PASSWORD"
	SecretFileMirrorReleaseRepoUsername = "AGENT_RELEASE_REPOSITORY_MIRROR_USERNAME"
	SecretFileMirrorReleaseRepoPassword = "AGENT_RELEASE_REPOSITORY_MIRROR_PASSWORD"
	SecretFileMirrorSharedRepoUsername  = "INSTANA_SHARED_REPOSITORY_MIRROR_USERNAME"
	SecretFileMirrorSharedRepoPassword  = "INSTANA_SHARED_REPOSITORY_MIRROR_PASSWORD"
	SecretFileHttpsProxy                = "HTTPS_PROXY"
)
