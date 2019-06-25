Install Using the Operator Lifecycle Manager (OLM)
--------------------------------------------------

The `instana-agent-operator` is available on [operator hub](https://operatorhub.io).

If you run the [Operator Lifecycle Manager (OLM)](https://github.com/operator-framework/operator-lifecycle-manager), you can install the `instana-operator-agent` as follows:

### Create the instana-agent namespace

Either click on _Administration_ -> _Namespaces_ -> _Create Namespace_ in the OLM user interface, or apply the following YAML:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: instana-agent
```

### Create the instana-agent operator group

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

### Create a Subscription

Create a Subscription for the Instana Agent Operator on the OLM's Web interface. Make sure to choose the `instana-agent` namespace.

This should start the `instana-agent-operator` Pod in the `instana-agent` namespace.

### Create an instana-agent resource

After creating the Subscription above, you will see the Instana Agent Operator under Installed Operators in the OLM's Web interface.

Click on _Create New_ to create a new `instana-agent` custom resource. You need to change some of the default values in the template. A description of the fields in the custom resource can be found in [install-manually.md](install-manually.md) under `instana-agent.customresource.yaml`.

Installing the custom resource should trigger the `instana-agent` Pods to be started in the `instana-agent` namespace. After a few minutes you should see your Kubernetes cluster in the Instana Web interface.
