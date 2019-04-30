Install Using the Operator Lifecycle Manager (OLM)
--------------------------------------------------

The `instana-agent-operator` is available on [operator hub](https://operatorhub.io). If you run the [Operator Lifecycle Manager (OLM)](https://github.com/operator-framework/operator-lifecycle-manager), you can install the `instana-operator-agent` by creating a subscription.

Before creating the subscription, you have to manually create a few resources:

* `instana-agent` namespace
* `instana-agent` operator group
* `agent-config` config map
* `config` config map

This document shows how to create these resources.

### instana-agent namespace

Either click on _Administration_ -> _Namespaces_ -> _Create Namespace_ in the OLM user interface, or apply the following YAML:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: instana-agent
```

### instana-agent operator group

Create a file `operator-group.yaml` with the following content:

```yaml
apiVersion: operators.coreos.com/v1alpha2
kind: OperatorGroup
metadata:
  name: instana-agent
  namespace: instana-agent
  spec:
    targetNamespaces:
    - instana-agent
```

Apply with `kubectl apply -f operator-group.yaml`

### agent-config config map

Create a file `agent-config.yaml` with the following content, or copy the file from the [deploy/](../deploy/) directory:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: agent-config
  namespace: instana-agent
data:
  configuration.yaml: |
    # You can leave this empty, or use this to configure your instana agent.
    # See https://docs.instana.io/quick_start/agent_setup/container/kubernetes/
```

Apply with `kubectl apply -f agent-config.yaml`.

### config config map

Create a file `config.yaml` with the following content, or copy the file from the [deploy/](../deploy/) directory:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: config
  namespace: instana-agent
data:
  zone.name: my-k8s-cluster
  agent.key: _PUT_YOUR_AGENT_KEY_HERE_
  agent.endpoint: saas-us-west-2.instana.io
  agent.endpoint.port: "443"
```

* Replace `my-k8s-cluster` with the with the cluster name that should be displayed in Instana
* Replace `_PUT_YOUR_AGENT_KEY_HERE_` with your Instana agent key

Apply with `kubectl apply -f config.yaml`.

### select namespace instana-agent

You are now done to create the subscription to the `instana-agent-operator`. Don't forget to select the `instana-agent` namespace when creating the subscription.
