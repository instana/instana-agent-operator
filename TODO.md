// TODO: Multiple zones -- when k8s_sensor is always in use?
// TODO: PodSecurityPolicy -- can this be dropped or wait until EOL on k8s/ocp versions that support it?
// TODO: Secrets and should sensitive data even be allowed in the CR?
// TODO: Add more logging (or events)
// TODO: status
// TODO: suite test with crud including deprectaed resource removal
// TODO: add to set of permissions needed by controller (CRUD owned resources + read CRD) + deletecollection
// TODO: exponential backoff config
// TODO: new ci build with all tests running + golangci lint, fix golangci settings
// TODO: fix "controller-manager" naming convention
// TODO: status and events (+conditions?)
// TODO: Update status as a defer in main reconcile at the top
// TODO: extra: auto set tolerations for different zones or for running on master, etc.
// TODO: Logger settings (rfc? / console?)
// TODO: more recovery ?

// TODO: extra: deprecate config_yaml string value and add a json.RawMessage version
// TODO: extra: Readiness probe for agent
// TODO: extra: Liveness and readiness probes for k8s sensor
// TODO: extra: Use startup probe on liveness / readiness checks?
// TODO: extra: additional read only volumes and other additional security constraints
// TODO: extra: PVs to save package downloads?
// TODO: extra: manage image refresh by restarting pods when update is available?
// TODO: extra: CRD validation flags (regex, jsonschema patterns, etc)?
// TODO: extra: runtime status from agents?
// TODO: extra: storage or ephemeral storage resource limits and requests?
// TODO: extra: cert generation when available?
// TODO: extra: inline resource (pod, etc.) config options?
// TODO: extra: Network policy usage, etc?

// TODO: extra: Possibly auto-detect zones via topology.kubernetes.io/zone label?
// TODO: extra: Toggle to tolerate masters automatically?