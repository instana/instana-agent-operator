package com.instana.operator;

import com.google.common.collect.ImmutableMap;
import com.instana.operator.customresource.InstanaAgent;
import com.instana.operator.customresource.InstanaAgentSpec;
import io.fabric8.kubernetes.api.model.Container;
import io.fabric8.kubernetes.api.model.EnvVar;
import io.fabric8.kubernetes.api.model.PodSpec;
import io.fabric8.kubernetes.api.model.apps.DaemonSet;
import io.fabric8.kubernetes.client.DefaultKubernetesClient;
import org.junit.jupiter.api.Test;

import static org.hamcrest.MatcherAssert.assertThat;
import static org.hamcrest.Matchers.allOf;
import static org.hamcrest.Matchers.hasItem;

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

    PodSpec podSpec = daemonSet.getSpec().getTemplate().getSpec();

    Container agentContainer = podSpec.getContainers().get(0);

    assertThat(agentContainer.getEnv(), allOf(
        hasItem(new EnvVar("INSTANA_AGENT_MODE", "APM", null))));
  }
}
