Installing the Instana Agent Operator Manually
==============================================

The [deploy/](../deploy) directory contains the YAML files that need to be applied to install the operator manually:

* `kubectl apply -f instana-agent.namespace.yaml`: Creates the `instana-agent` namespace.
* `kubectl apply -f instana-agent.serviceaccount.yaml`: Creates the `instana-agent` service account.
* `kubectl apply -f instana-agent.clusterrole.yaml`: Creates the `instana-agent` cluster role.
* `kubectl apply -f instana-agent.clusterrolebinding.yaml`: Creates the `instana-agent` cluster role binding.
* `kubectl apply -f instana-agent.crd.yaml`: Creates the `instana-agent` custom resource definition.

Now edit `instana-agent.customresource.yaml` and replace at least the following values:

  * `agent.key` must be set with your Instana agent key.
  * `agent.endpoint` must be set with the monitoring ingress endpoint, generally either `saas-us-west-2.instana.io` or `saas-eu-west-1.instana.io`.
  * `agent.endpoint.port` must be set with the monitoring ingress port, generally `"443"` (wrapped in quotes).
  * `agent.zone.name` should be set with the name of the Kubernetes cluster that is be displayed in Instana.

In case you need to adapt `configuration.yaml`, view the documentation here: [https://docs.instana.io/quick_start/agent_setup/container/kubernetes/](https://docs.instana.io/quick_start/agent_setup/container/kubernetes/).

Finally, deploy the custom resource and the operator:

* `kubectl apply -f agent-config.customresource.yaml`: Creates the `instana-agent` custom resource.
* `kubectl apply -f instana-agent-operator.deployment.yaml`: Creates the `instana-agent-operator` deployment.
