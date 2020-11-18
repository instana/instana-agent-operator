package com.instana.operator;

import static com.instana.operator.env.Environment.RELATED_IMAGE_INSTANA_AGENT;
import static java.net.HttpURLConnection.HTTP_OK;
import static java.util.Collections.emptyMap;
import static org.hamcrest.MatcherAssert.assertThat;
import static org.hamcrest.Matchers.allOf;
import static org.hamcrest.Matchers.hasEntry;
import static org.hamcrest.Matchers.hasItem;
import static org.hamcrest.Matchers.hasKey;
import static org.hamcrest.Matchers.is;
import static org.hamcrest.Matchers.not;

import java.util.Collections;
import java.util.Map;

import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import com.google.common.collect.ImmutableMap;
import com.instana.operator.customresource.InstanaAgent;
import com.instana.operator.customresource.InstanaAgentSpec;
import com.instana.operator.env.Environment;

import io.fabric8.kubernetes.api.model.Container;
import io.fabric8.kubernetes.api.model.EnvVar;
import io.fabric8.kubernetes.api.model.apiextensions.CustomResourceDefinitionBuilder;
import io.fabric8.kubernetes.api.model.apps.DaemonSet;
import io.fabric8.kubernetes.client.DefaultKubernetesClient;
import io.fabric8.kubernetes.client.server.mock.KubernetesMockServer;

public class AgentDeployerTest {

  private DefaultKubernetesClient client;
  private KubernetesMockServer server;

  @BeforeEach
  public void setup() {
    server = new KubernetesMockServer();
    server.start();
    client = (DefaultKubernetesClient) server.createClient();
  }

  @AfterEach
  public void teardown() {
    server.shutdown();
  }

  @Test
  void daemonset_must_include_environment() {
    AgentDeployer deployer = new AgentDeployer();
    deployer.setDefaultClient(client);
    deployer.setEnvironment(empty());

    InstanaAgentSpec agentSpec = new InstanaAgentSpec();
    agentSpec.setAgentEnv(ImmutableMap.<String, String>builder()
        .put("INSTANA_AGENT_MODE", "APM")
        .build());

    InstanaAgent customResource = new InstanaAgent();
    customResource.setSpec(agentSpec);

    DaemonSet daemonSet = deployer.newDaemonSet(
        customResource,
        client.inNamespace("instana-agent").apps().daemonSets());

    Container agentContainer = getAgentContainer(daemonSet);

    assertThat(agentContainer.getEnv(), allOf(
        hasItem(new EnvVar("INSTANA_AGENT_MODE", "APM", null))));
  }

  @Test
  void daemonset_must_include_specified_image() {
    AgentDeployer deployer = new AgentDeployer();
    deployer.setDefaultClient(client);
    deployer.setEnvironment(empty());

    InstanaAgentSpec agentSpec = new InstanaAgentSpec();
    agentSpec.setAgentImage("other/image:some-tag");

    InstanaAgent customResource = new InstanaAgent();
    customResource.setSpec(agentSpec);

    DaemonSet daemonSet = deployer.newDaemonSet(
        customResource,
        client.inNamespace("instana-agent").apps().daemonSets());

    Container agentContainer = getAgentContainer(daemonSet);

    assertThat(agentContainer
        .getImage(), is("other/image:some-tag"));
  }

  @Test
  void daemonset_must_include_image_from_csv_if_specified() {
    AgentDeployer deployer = new AgentDeployer();
    deployer.setDefaultClient(client);
    deployer.setEnvironment(singleVar(RELATED_IMAGE_INSTANA_AGENT, "other/image:some-tag"));

    InstanaAgent customResource = new InstanaAgent();
    customResource.setSpec(new InstanaAgentSpec());

    DaemonSet daemonSet = deployer.newDaemonSet(
        customResource,
        client.inNamespace("instana-agent").apps().daemonSets());

    Container agentContainer = getAgentContainer(daemonSet);

    assertThat(agentContainer.getImage(), is("other/image:some-tag"));
  }

  @Test
  public void daemonset_must_include_version_label_if_specified_on_crd() {
    server
        .expect()
        .get()
        .withPath("/apis/apiextensions.k8s.io/v1beta1/customresourcedefinitions/agents.instana.io")
        .andReturn(HTTP_OK, new CustomResourceDefinitionBuilder()
                  .withNewMetadata()
                  .withName("agents.instana.io")
                  .withLabels(ImmutableMap.of("app.kubernetes.io/version", "0.3.8"))
                  .endMetadata()
                  .build())
        .always();

    AgentDeployer deployer = new AgentDeployer();
    deployer.setDefaultClient(client);
    deployer.setEnvironment(empty());

    InstanaAgent customResource = new InstanaAgent();
    customResource.setSpec(new InstanaAgentSpec());

    DaemonSet daemonSet = deployer.newDaemonSet(
        customResource,
        client.inNamespace("instana-agent").apps().daemonSets());

    Map<String, String> labels = daemonSet.getMetadata().getLabels();
    assertThat(labels, hasEntry(is("app.kubernetes.io/managed-by"), is("instana-agent-operator")));
    assertThat(labels, hasEntry(is("app.kubernetes.io/version"), is("0.3.8")));
  }

  @Test
  public void daemonset_must_not_include_version_label_if_not_specified_on_crd() {
    server
        .expect()
        .get()
        .withPath("/apis/apiextensions.k8s.io/v1beta1/customresourcedefinitions/agents.instana.io")
        .andReturn(HTTP_OK, new CustomResourceDefinitionBuilder()
            .withNewMetadata()
            .withName("agents.instana.io")
            .endMetadata()
            .build())
        .always();

    AgentDeployer deployer = new AgentDeployer();
    deployer.setDefaultClient(client);
    deployer.setEnvironment(empty());

    InstanaAgent customResource = new InstanaAgent();
    customResource.setSpec(new InstanaAgentSpec());

    DaemonSet daemonSet = deployer.newDaemonSet(
        customResource,
        client.inNamespace("instana-agent").apps().daemonSets());

    Map<String, String> labels = daemonSet.getMetadata().getLabels();
    assertThat(labels, hasEntry(is("app.kubernetes.io/managed-by"), is("instana-agent-operator")));
    assertThat(labels, not(hasKey("app.kubernetes.io/version")));
  }

  private Container getAgentContainer(DaemonSet daemonSet) {
    return daemonSet.getSpec().getTemplate().getSpec().getContainers().get(0);
  }

  private Environment empty() {
    return Environment.fromMap(emptyMap());
  }

  private Environment singleVar(String key, String value) {
    return Environment.fromMap(Collections.singletonMap(key, value));
  }
}
