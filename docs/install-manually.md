Installing the Instana Agent Operator Manually
==============================================

The [deploy/](../deploy) directory contains the YAML files that need to be applied to install the operator manually:

* `kubectl apply -f instana-agent.namespace.yaml`: Creates the `instana-agent` namespace.
* `kubectl apply -f instana-agent.serviceaccount.yaml`: Creates the `instana-agent` service account.
* `kubectl apply -f instana-agent.clusterrole.yaml`: Creates the `instana-agent` cluster role.
* `kubectl apply -f instana-agent.clusterrolebinding.yaml`: Creates the `instana-agent` cluster role binding.
* `kubectl apply -f electedleader.crd.yaml`: Creates the `electedleader` custom resource definition.
* `kubectl apply -f config.configmap.yaml`: Creates the `config` config map with general needed Agent settings. Before
  running this command, you need to replace at least the following values in `config.configmap.yaml`:
  * `agent.key` must be set with your Instana agent key.
  * `agent.endpoint` must be set with the monitoring ingress endpoint, generally either `saas-us-west-2.instana.io` or `saas-eu-west-1.instana.io`.
  * `agent.endpoint.port` must be set with the monitoring ingress port, generally `"443"` (wrapped in quotes).
* `kubectl apply -f agent-config.configmap.yaml`: Creates the `agent-config` config map with other optional Agent settings.
  Documentation on the available configuration options can be found on [https://docs.instana.io/quick_start/agent_setup/container/kubernetes/](https://docs.instana.io/quick_start/agent_setup/container/kubernetes/).
* `kubectl apply -f instana-agent-operator.deployment.yaml`: Creates the `instana-agent-operator` deployment.

After applying the files in this order, you should see an `instana-agent-operator` pod running in the `instana-agent` namespace. The operator will deploy a daemon set running an `instana-agent` Pod on each Kubernetes node.
