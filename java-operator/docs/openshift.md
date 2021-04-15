Prerequisites for installing on OpenShift
--------------------------------------------------

Before the agent will be able to run in OpenShift, you need to perform a couple of extra configuration steps.

You need to set up a project for the Instana Agent and configure it's permissions.

The project you create here needs to be the namespace where you create the Instana Agent custom resource that the operator will use to deploy the agent.

For example, create the `instana-agent` project:

    oc new-project instana-agent

Then, ensure the `instana-agent` service account is in the privileged security context:

    oc adm policy add-scc-to-user privileged -z instana-agent

This service account will be created by the operator.

Now you can proceed with installing the operator for the Instana agent.
