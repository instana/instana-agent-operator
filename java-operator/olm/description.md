# Instana

Instana is an [APM solution](https://www.instana.com/product-overview/) built for microservices that enables IT Ops to build applications faster and deliver higher quality services by automating monitoring, tracing and root cause analysis. The solution is optimized for [Kubernetes](https://www.instana.com/automatic-kubernetes-monitoring/) and [OpenShift](https://www.instana.com/blog/automatic-root-cause-analysis-for-openshift-applications/).

## Instana Agent Operator

This is the Kubernetes Operator for installing the Instana Agent on Kubernetes or OpenShift.

## Prerequisites for OpenShift

Before the agent will be able to run in OpenShift, you need to perform a couple of extra configuration steps.

You need to set up a project for the Instana Agent and configure it's permissions.

The project you create here needs to be the namespace where you create the Instana Agent custom resource that the operator will use to deploy the agent.

For example, create the `instana-agent` project:

    oc new-project instana-agent

Then, ensure the `instana-agent` service account is in the privileged security context:

    oc adm policy add-scc-to-user privileged -z instana-agent

This service account will be created by the operator.

Now you can proceed with installing the operator for the Instana agent.

## Installation and Configuration

First, install this operator from [OperatorHub.io](https://operatorhub.io/), [OpenShift Container Platform](https://www.openshift.com/), or [OKD](https://www.okd.io/).

Second, create the target namespace where the Instana agent should be installed. The agent does not need to run in the same namespace as the operator. Most users create a new namespace `instana-agent` for running the agents.

Third, create a custom resource with the agent configuration in the target namespace. The operator will pick up the custom resource and install the Instana agent accordingly.

The following is a minimal template of the custom resource:

```yaml
apiVersion: instana.io/v1beta1
kind: InstanaAgent
metadata:
  name: instana-agent
  namespace: instana-agent
spec:
  agent.zone.name: my-zone # (optional) name of the zone of the host
  agent.key: replace-me # replace with your Instana agent key
  agent.endpoint.host: ingress-red-saas.instana.io # the monitoring ingress endpoint
  agent.endpoint.port: 443 # the monitoring ingress endpoint port, wrapped in quotes
  agent.env:
    INSTANA_AGENT_TAGS: example
  cluster.name: replace-me # replace with the name of your Kubernetes cluster
  config.files:
    configuration.yaml: |
      # You can leave this empty, or use this to configure your instana agent.
      # See https://docs.instana.io/setup_and_manage/host_agent/on/kubernetes/
```

Save the template in a file `instana-agent.yaml` and edit the following values:

* If your target namespace is not `instana-agent`, replace the `namespace:` accordingly.
* `agent.key` must be set with your Instana agent key.
* `agent.endpoint` must be set with the monitoring ingress endpoint, generally either `saas-us-west-2.instana.io` or `saas-eu-west-1.instana.io`.
* `agent.endpoint.port` must be set with the monitoring ingress port, generally "443" (wrapped in quotes).
* `agent.zone.name` should be set with the name of the Kubernetes cluster that is be displayed in Instana.

For advanced configuration, you can edit the contents of the `configuration.yaml` file. View documentation [here](https://docs.instana.io/setup_and_manage/host_agent/on/kubernetes/).

Apply the custom resource with `kubectl apply -f instana-agent.yaml`. After some time, you should see `instana-agent` Pods being created on each node of your cluster, and your cluster should show on the infrastructure map on your Instana Web interface.

## Uninstalling

In order to uninstall the Instana agent, simply remove the custom resource with `kubectl delete -f instana-agent.yaml`.

## Source Code

The Instana agent operator is an open source project hosted on [https://github.com/instana/instana-agent-operator](https://github.com/instana/instana-agent-operator/).
