/*
(c) Copyright IBM Corp. 2024
(c) Copyright Instana Inc. 2024
*/

package namespaces

// File content to mount into the agent daemonset at /opt/instana/agent/etc/namespaces/namespaces.yaml (provided as env var)
// An empty file will look like below
/*
version: 1
namespaces: {}
*/
// If a namespace carried the label, the file will provide this yaml. Only the label instana-workload-monitoring is stored, not every label
/*
version: 1
namespaces:
    instana-agent:
        labels:
            instana-workload-monitoring: "true"
*/
type NamespacesDetails struct {
	Version    int                          `json:"version"`
	Namespaces map[string]NamespaceMetadata `json:"namespaces"`
}

type NamespaceMetadata struct {
	Labels map[string]string `json:"labels"`
}
