## Next Steps

## Error Wrapping

Custom error types should be created with relevant messages to wrap errors that are being passed up the stack with
relevant information for debugging.

### ~~Fix K8Sensor Deployment And~~ Test Seamless Upgrade from deprecated k8s sensors

~~Deployment of K8Sensor is currently broken, fix this and then~~ run tests to ensure upgrade from a configuration
using the deprecated kubernetes sensor is seamless.

### Misconfiguration Errors

If the user misconfigures the agent then the attempt to create or update the Agent CR should be rejected. This can be
achieved using one of or a combination of the following methods.

#### CR Validation

[Validation rules](https://kubernetes.io/blog/2022/09/23/crd-validation-rules-beta/) and schema-based
[generation](https://book.kubebuilder.io/reference/markers/crd.html),
[validation](https://book.kubebuilder.io/reference/markers/crd-validation.html), and
[processing](https://book.kubebuilder.io/reference/markers/crd-processing.html) rules can be used to verify validity of
user-provided configuration and provide useful feedback for troubleshooting.

#### Webhook Validation

[Defaulting and Validation Webhooks](https://book.kubebuilder.io/cronjob-tutorial/webhook-implementation) could be used
for more advanced validation and to ensure defaulted values will appear on the CR present on the cluster without the
need for updates to the CR by the controller that could cause performance issues if another controller is managing the
agent CR.

#### Validation Admission Policy

Beginning in k8s v1.26 (alpha) or v1.28 (beta) a
[ValidationAdmissionPolicy](https://kubernetes.io/docs/reference/access-authn-authz/validating-admission-policy/) may
also be used to configure validation rules using [Common Expression Language (CEL)](https://github.com/google/cel-spec).

### Testing

In addition to any updates that will be made to the end-to-end tests, behavioral tests related to installation,
uninstallation, and upgrade (including upgrade from agent v2 to v3) should be written in
[controllers/suite_test.go](./controllers/suite_test.go). These will be able to verify most changes to the operator's
behavior, and they will run (and fail) much more quickly than the e2e tests. There are also a few gaps in the current
unit testing coverage that should ideally be filled. There are also a few broken tests as a result of changes made
after they were written that need to be fixed.

### CI Updates

The operator build should run all unit tests, behavioral tests, and [static code linting](.golangci.yml). Code linting
settings should be reviewed and configured. It may also be useful to have PR builds that will automatically regenerate
manifests and bundle YAMLs and commit the changes to the source branch as well as running the same tests and linting as
release branches.

### Chart Update

The Helm chart should be updated to wrap the operator and an instance of the CR built by directly using toYaml on the
Values.yaml file to construct the spec.

#### Chart Update Automation

Some automation to make updates to the chart based on changes to the CRD or
operator deployment may be desirable if these things are expected to change often.

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
(e.g. --zap-time-encoding=rfc3339 --zap-encoder=console).

### .spec.agent.configuration_yaml

This could potentially be deprecated and replaced by a field using the `json.RawMessage` type, which would enable the
configuration yaml to be configured using native yaml within the CR rather than as an embedded string.

### Probes

The agent should have a readiness probe in addition to its liveness probe. The k8s_sensor should also have liveness and
readiness probes. A startup probe can also be added to the agent now that it is supported in all stable versions of k8s.
This will allow for faster readiness and recovery from soft-locks (if they occur) since it will allow the
initialDelaySeconds to be reduced on the liveness probe. The agent and k8sensor may also want to create dedicated
readiness endpoints to allow their actual availability to be reflected more accurately in the CR status. In the
agent's case, the availability of independent servers running on different ports may need to be considered when
deciding whether to do this since traffic directed at k8s services will not be forwarded to pods that are marked as
unready.

### PVs For Package Downloads

Optional Persistent volumes could potentially be used to cache dynamically downloaded updates and packages in between
agent restarts.

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

### Auto-Reload on Agent-Key or Download-Key Change

Currently, the agent-key and download-key are read by the agent via environment variable set via referencing a key in
one or more k8s secrets. It would be beneficial to watch the secret(s) and trigger a restart of the agent daemsonset if
a change is detected in the secret(s).