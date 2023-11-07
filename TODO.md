## Next Steps

### Template Files

The current configuration of the agent daemonset directly overloads the /opt/instana/agent/etc/instana directory in
order to allow for direct hot-reload of configuration files from the configmap. This blocks some of the templating
that normally gets run by the run.sh script upon start up. If needed this can be replaced by implementing equivalent
behavior within the operator such that the templated files will be generated and placed into the agent ConfigMap.

#### Update

The agent can be updated to scan yamls from a non-default directory when an environment variable is set
(e.g. INSTANA_CONFIG_DIR). The operator should then be configured to hardcode the value of this environment variable in
the agent pod to the directory where the agent configmap will get mounted which should be different from the default
location (e.g. /opt/instana/agent/etc/instana-k8s).

### Agent Mode

.spec.agent.mode needs to be set to KUBERNETES by default and should disallow (or at least warn against) any modes
that use the deprecated version of the k8s-sensor that runs within the agent.

**Edit:** Need to determine how to properly configure the k8s sensor since the
current configuration does not seem to be correct.

### Multi-Zone Support

Options for deployment across multiple zones should be enabled. We will need to determine what steps need to be taken to
support this configuration when using the k8s_sensor.

### PodSecurityPolicy

The PodSecurityPolicy may need to be constructed and deployed in certain cases still. It has been removed from all
currently supported versions of vanilla k8s except v1.24 which will reach EOL on 28 Jul 2023, but is still included in
OCP 4.10 and 4.11, though it is deprecated in these versions and typically OpenShift's SecurityContextConstraint
resource is used in place of PodSecurityPolicy in OCP.

### Logging and Events

Additional logging needs to be added to the new operator code to log successful operations as they occur. In some cases
we may also wish to produce [events](https://kubernetes.io/docs/reference/kubernetes-api/cluster-resources/event-v1/)
into k8s for our agent CR.

### Status

Code should be added to populate the agent status field to replicate the status fields tracked by the previous version
of the operator. This should be done in a "defer" to ensure status will always be updated in the event of an error
during deployment. In the future we may wish to deprecate existing status fields and replace them with more
[standardized](https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Condition) status fields.

### Testing

In addition to any updates that will be made to the end-to-end tests, behavioral tests related to installation,
uninstallation, and upgrade (including upgrade from agent v2 to v3) should be written in
[controllers/suite_test.go](./controllers/suite_test.go). These will be able to verify most changes to the operator's
behavior, and they will run (and fail) much more quickly than the e2e tests. There are also a few gaps in the current
unit testing coverage that should ideally be filled. There are also a few broken tests as a result of changes made
after they were written that need to be fixed.

### Operator Permissions

The permissions required by the operator will need to be updated to include any new permissions required by the new
version of the operator. Preferably these should be included as comments in
[controllers/instanaagent_controller.go](./controllers/instanaagent_controller.go), so that the appropriate k8s
manifests can be generated automatically. Additional needed permissions include get/list/watch access to
CustomResourceDefinitions and create/update/patch/delete/get/list/watch for all types of resources directly owned by
the agent that are not already included in the existing permission set.

### CI Updates

The operator build should run all unit tests, behavioral tests, and [static code linting](.golangci.yml). Code linting
settings should be reviewed and configured. It may also be useful to have PR builds that will automatically regenerate
manifests and bundle YAMLs and commit the changes to the source branch as well as running the same tests and linting as
release branches.

### Operator Naming Convention

The "controller-manager" naming convention should be replaced by something unique (ie instana-agent-operator) to ensure
that it is clear which resources belong to us and to ensure there are no conflicts with other operators since
"controller-manager" is the default value that is generated when operator projects are initialized.

### Chart Update

The Helm chart should be updated to wrap the operator and an instance of the CR built by directly using toYaml on the
Values.yaml file to construct the spec.

## Future Considerations

### Sensitive Data

Currently sensitive data (agent key, download, key, certificates, etc.) can be configured directly with the Agent CR.
This is considered bad-practice and can be a security risk in some cases; however, it may be alright to keep as a means
to deploy agents easily in development environments. Customers should be advised to place sensitive data directly into
secrets and reference the secrets from the agent spec.

### Configure Exponential Backoff

Rate-limiting [for the controller](https://danielmangum.com/posts/controller-runtime-client-go-rate-limiting/) should
be configured to prevent potential performance issues in cases where the cluster is inaccessible or the agent otherwise
cannot be deployed for some reason.

### Automatic Tolerations

Options could be added to the CR-spec to enable agents to run on master nodes by automatically setting the appropriate
tolerations for node taints.

### Automatic Zones

If desired an option could be added to automatically assign zone names to agent instances based on the value of the
`topology.kubernetes.io/zone` label on the node on which they are running.

### Logging

It may be worth considering the use of different default logging settings to improve readability
(eg. --zap-time-encoding=rfc3339 --zap-encoder=console).

### .spec.agent.configuration_yaml

This could potentially be deprecated and replaced by a field using the `json.RawMessage` type, which would enable the
configuration yaml to be configured using native yaml within the CR rather than as an embedded string.

### Probes

The agent should have a readiness probe in addition to its liveness probe. The k8s_sensor should also have liveness and
readiness probes. A startup probe can also be added to the agent now that it is supported in all stable versions of k8s.
This will allow for faster readiness and recovery from soft-locks (if they occur) since it will allow the
initialDelaySeconds to be reduced on the liveness probe.

### PVs For Package Downloads

Optional Persistent volumes could potentially be used to cache dynamically downloaded updates and packages in between
agent restarts.

### CR Validation

[Validation rules](https://kubernetes.io/blog/2022/09/23/crd-validation-rules-beta/) and schema-based
[generation](https://book.kubebuilder.io/reference/markers/crd.html),
[validation](https://book.kubebuilder.io/reference/markers/crd-validation.html), and
[processing](https://book.kubebuilder.io/reference/markers/crd-processing.html) rules can be used to verify validity of
user-provided configuration and provide useful feedback for troubleshooting.

### Runtime Status

Runtime status information from the agent could be scraped and incorporated into the status tracked by the CR if this
is deemed useful.

### Ephemeral Storage Requests/Limits

Requests and limits for
[ephemeral storage](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#setting-requests-and-limits-for-local-ephemeral-storage)
can be set to ensure that agent pods are assigned to nodes containing appropriate storage space for dynamic agent
package downloads or to ensure agents do not exceed some limit of storage use on the host node.

### Certificate Generation

If desired, certificates could be automatically generated and configured when appropriate if cert-manager or
OpenShift's certificate generation is available.

### Network Policies

[Network policies](https://kubernetes.io/docs/concepts/services-networking/network-policies/) can be used to restrict
inbound traffic on ports that the agent or k8s_sensor do not use as a security measure. (May not work on agent itself
due to `hostNetwork: true).