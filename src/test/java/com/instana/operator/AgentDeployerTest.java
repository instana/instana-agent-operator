package com.instana.operator;

import com.google.common.collect.ImmutableMap;
import com.instana.operator.customresource.InstanaAgent;
import com.instana.operator.customresource.InstanaAgentSpec;
import io.fabric8.kubernetes.api.model.Container;
import io.fabric8.kubernetes.api.model.EnvVar;
import io.fabric8.kubernetes.api.model.apps.DaemonSet;
import io.fabric8.kubernetes.client.DefaultKubernetesClient;
import org.junit.jupiter.api.Test;

import java.util.HashMap;

import static com.instana.operator.env.Environment.RELATED_IMAGE_INSTANA_AGENT;
import static org.hamcrest.MatcherAssert.assertThat;
import static org.hamcrest.Matchers.allOf;
import static org.hamcrest.Matchers.hasItem;
import static org.hamcrest.Matchers.is;

class AgentDeployerTest {

  private final DefaultKubernetesClient client = new DefaultKubernetesClient();

  @Test
  void daemonset_must_include_environment() {
    AgentDeployer deployer = new AgentDeployer();

    InstanaAgentSpec agentSpec = new InstanaAgentSpec();
    agentSpec.setAgentEnv(ImmutableMap.<String, String>builder()
        .put("INSTANA_AGENT_MODE", "APM")
        .build());

    InstanaAgent crd = new InstanaAgent();
    crd.setSpec(agentSpec);

    DaemonSet daemonSet = deployer.newDaemonSet(
        crd,
        client.inNamespace("instana-agent").apps().daemonSets());

    Container agentContainer = getAgentContainer(daemonSet);

    assertThat(agentContainer.getEnv(), allOf(
        hasItem(new EnvVar("INSTANA_AGENT_MODE", "APM", null))));
  }

  @Test
  void daemonset_must_include_specified_image() {
    AgentDeployer deployer = new AgentDeployer();

    InstanaAgentSpec agentSpec = new InstanaAgentSpec();
    agentSpec.setAgentImage("other/image:some-tag");

    InstanaAgent crd = new InstanaAgent();
    crd.setSpec(agentSpec);

    DaemonSet daemonSet = deployer.newDaemonSet(
        crd,
        client.inNamespace("instana-agent").apps().daemonSets());

    Container agentContainer = getAgentContainer(daemonSet);

    assertThat(agentContainer
        .getImage(), is("other/image:some-tag"));
  }

  @Test
  void daemonset_must_include_image_from_csv_if_specified() {
    HashMap<String, String> environment = new HashMap<>();
    AgentDeployer deployer = new AgentDeployer();
    deployer.setEnvironment(environment::get);

    environment.put(RELATED_IMAGE_INSTANA_AGENT, "other/image:some-tag");

    InstanaAgent crd = new InstanaAgent();
    crd.setSpec(new InstanaAgentSpec());

    DaemonSet daemonSet = deployer.newDaemonSet(
        crd,
        client.inNamespace("instana-agent").apps().daemonSets());

    Container agentContainer = getAgentContainer(daemonSet);

    assertThat(agentContainer.getImage(), is("other/image:some-tag"));
  }

  private Container getAgentContainer(DaemonSet daemonSet) {
    return daemonSet.getSpec().getTemplate().getSpec().getContainers().get(0);
  }
}
