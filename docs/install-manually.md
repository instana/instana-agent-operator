Installing the Instana Agent Operator Manually
==============================================

### Prerequisites for OpenShift

Before you start, if you are installing the operator into OpenShift, note the extra steps required [here](openshift.md).

### Steps

Deploy the operator as follows:

```bash
kubectl apply -f https://raw.githubusercontent.com/instana/instana-agent-operator/master/olm/operator-resources/instana-agent-operator.yaml
```

Now the operator should be up and running in namespace `instana-agent`, waiting for an `instana-agent` custom resource to
be created.

Create the custom resource yaml file, following [this](https://github.com/instana/instana-agent-operator/blob/master/deploy/instana-agent.customresource.yaml) template.

Edit the template and replace at least the following values:

  * `agent.key` must be set with your Instana agent key.
  * `agent.endpoint` must be set with the monitoring ingress endpoint, generally either `saas-us-west-2.instana.io` or `saas-eu-west-1.instana.io`.
  * `agent.endpoint.port` must be set with the monitoring ingress port, generally `"443"` (wrapped in quotes).
  * `agent.zone.name` should be set with the name of the Kubernetes cluster that is be displayed in Instana.
  * `agent.env` can be used to specify environment variables for the agent, for instance, proxy configuration. See possible environment values [here](https://docs.instana.io/quick_start/agent_setup/container/docker/). For instance:

        agent.env:
          INSTANA_AGENT_TAGS: staging

  * `config.files` can be used to specify configuration files, for instance, specifying a `configuration.yaml`:

        config.files:
          configuration.yaml: |
            # Example of configuration yaml template

            # Host
            #com.instana.plugin.host:
            #  tags:
            #    - 'dev'
            #    - 'app1'

     In case you need to adapt `configuration.yaml`, view the documentation here: [https://docs.instana.io/quick_start/agent_setup/container/kubernetes/](https://docs.instana.io/quick_start/agent_setup/container/kubernetes/).

Apply the edited custom resource:

```bash
kubectl apply -f instana-agent.customresource.yaml
```

The operator will pick up the configuration from the custom resource and deploy the Instana agent.
