/*
(c) Copyright IBM Corp. 2024
(c) Copyright Instana Inc. 2024
*/

package namespaces

// File content to mount into the agent daemonset
/*
	{
	  "version": 1,
	  "namespaces": {
	    "namespaceName1": {
	      "labels": {
	        "key1": "value1",
	        "key2": "value2"
	      }
	    },
	    "namespaceName2": {
	      "labels": {
	        "key3": "value3",
	        "key4": "value4"
	      }
	    }
	  }
	}
*/
type NamespacesDetails struct {
	Version    int                          `json:"version"`
	Namespaces map[string]NamespaceMetadata `json:"namespaces"`
}

type NamespaceMetadata struct {
	Labels map[string]string `json:"labels"`
}
