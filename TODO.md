// TODO: Remember to feed logger into context

// TODO: apply all function, basic controller tasks
// TODO: then status later on, suite test with crud including deprectaed resource removal
// TODO: add to set of permissions needed by controller (CRUD owned resources + read CRD) + deletecollection
// TODO: Delete resources with previous gen label for all possible dependent types after successful apply
// TODO: Use CM to track dependents and use set diff to delete old ones

// TODO: ~~warning (error) if not expected name and namespace (and status/event?)~~ -> shouldn't be needed with helm
uninstall logic below
// TODO: Keep Helm uninstall step for migration -> Do this step iff (old) finalizer is present as this indicates
upgrade, use a different finalizer from now on to track cluster-scoped resources
// TODO: owned resources in controller watch
// TODO: exponential backoff config
// TODO: new ci build with all tests running + golangci lint, fix golangci settings
// TODO: fix "controller-manager" naming convention
// TODO: status and events (+conditions?)
// TODO: Update status as a defer in main reconcile function just above apply logic
// TODO: extra auto detect OpenShift, auto set tolerations, etc.
// TODO: finalizers to delete cluster-scoped resource types via deletecollection on labels?
// TODO: Logger settings
// TODO: Recovery somewhere?

// TODO: deprecation config_yaml string value and add a json.RawMessage version
// TODO: Validation webhook or just status errors?
// TODO: Readiness probe
// TODO: Use startup probe on liveness / readiness checks?
// TODO: additional read only volumes and other additional security constraints
// TODO: PVs to save package downloads?
// TODO: manage image refresh by restarting pods when update is available?
// TODO: CRD validation flags (regex, jsonschema patterns, etc)?
// TODO: extra: runtime status from agents?
// TODO: storage or ephemeral storage resource limits and requests?
// TODO: cert generation when available?
// TODO: inline resource (pod, etc.) config options?
// TODO: Network policy usage, etc?

// TODO: Possibly auto-detect zones via topology.kubernetes.io/zone label?
// TODO: Toggle to tolerate masters automatically?