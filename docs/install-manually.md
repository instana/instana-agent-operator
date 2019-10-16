Installing the Instana Agent Operator Manually
==============================================

Before you start, if you are installing the operator into OpenShift, note the extra steps required [here](openshift.md).

The [deploy/](../deploy) directory contains the YAML files that need to be applied to install the operator manually.
Deploy the operator as follows:

```bash
kubectl apply -f instana-agent-operator.yaml
```

Now the operator should be up and running in namespace `instana-agent`, waiting for an `instana-agent` custom resource to
be created. Before creating the custom resource, you must edit `instana-agent.customresource.yaml` and replace at least the following values:

  * `agent.key` must be set with your Instana agent key.
  * `agent.endpoint` must be set with the monitoring ingress endpoint, generally either `saas-us-west-2.instana.io` or `saas-eu-west-1.instana.io`.
  * `agent.endpoint.port` must be set with the monitoring ingress port, generally `"443"` (wrapped in quotes).
  * `agent.zone.name` should be set with the name of the Kubernetes cluster that is be displayed in Instana.

In case you need to adapt `configuration.yaml`, view the documentation here: [https://docs.instana.io/quick_start/agent_setup/container/kubernetes/](https://docs.instana.io/quick_start/agent_setup/container/kubernetes/).

Apply the edited custom resource:

```bash
kubectl apply -f instana-agent.customresource.yaml
```

The operator will pick up the configuration from the custom resource and deploy the Instana agent.
