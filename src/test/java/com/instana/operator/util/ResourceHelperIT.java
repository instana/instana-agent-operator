package com.instana.operator.util;

import static com.instana.operator.util.ConfigUtils.createClientConfig;
import static com.instana.operator.util.OkHttpClientUtils.createHttpClient;
import static org.hamcrest.MatcherAssert.assertThat;

import java.util.UUID;

import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.dataformat.yaml.YAMLMapper;

import io.fabric8.kubernetes.api.model.ConfigMap;
import io.fabric8.kubernetes.api.model.Namespace;
import io.fabric8.kubernetes.api.model.Secret;
import io.fabric8.kubernetes.api.model.ServiceAccount;
import io.fabric8.kubernetes.api.model.apps.DaemonSet;
import io.fabric8.kubernetes.api.model.rbac.ClusterRole;
import io.fabric8.kubernetes.api.model.rbac.ClusterRoleBinding;
import io.fabric8.kubernetes.client.Config;
import io.fabric8.kubernetes.client.DefaultKubernetesClient;

class ResourceHelperIT {

  YAMLMapper mapper;
  DefaultKubernetesClient client;
  Namespace agentNS;
  String uuid;
  String name;

  @BeforeEach
  void setUp() throws Exception {
    Config config = createClientConfig();
    client = new DefaultKubernetesClient(createHttpClient(config), config);
    agentNS = client.namespaces().createNew()
        .withNewMetadata()
        .withName(UUID.randomUUID().toString())
        .endMetadata()
        .done();
    uuid = agentNS.getMetadata().getName();
    name = "instana-agent";

    mapper = new YAMLMapper();
  }

  @AfterEach
  void cleanUp() {
    //client.namespaces().delete(agentNS);
  }

  @Test
  void canCreateDaemonSet() throws JsonProcessingException, InterruptedException {
    // ServiceAccount
    ServiceAccount svc = AgentResourcesUtil.createServiceAccount(uuid, name, null);
    System.out.println(mapper.writeValueAsString(svc));
    assertThat("serviceaccount was created", null != client.serviceAccounts().create(svc).getMetadata().getUid());

    // Secret
    Secret s = AgentResourcesUtil.createAgentKeySecret(uuid, name, System.getenv("INSTANA_AGENT_KEY"), null);
    System.out.println(mapper.writeValueAsString(s));
    assertThat("secret was created", null != client.secrets().create(s).getMetadata().getUid());

    // ConfigMap
    ConfigMap m = AgentResourcesUtil.createConfigurationConfigMap(uuid, name, null);
    System.out.println(mapper.writeValueAsString(m));
    assertThat("configmap was created", null != client.configMaps().create(m).getMetadata().getUid());

    // ClusterRole
    ClusterRole r = AgentResourcesUtil.createAgentClusterRole(name, null);
    System.out.println(mapper.writeValueAsString(r));
    assertThat("clusterrole was created",
        null != client.rbac().clusterRoles().createOrReplace(r).getMetadata().getUid());

    // ClusterRoleBinding
    ClusterRoleBinding rb = AgentResourcesUtil.createAgentClusterRoleBinding(uuid, name, svc, r, null);
    System.out.println(mapper.writeValueAsString(r));
    assertThat("clusterrolebinding was created",
        null != client.rbac().clusterRoleBindings().createOrReplace(rb).getMetadata().getUid());

    // DaemonSet
    DaemonSet ds = AgentResourcesUtil.createAgentDaemonSet(
        uuid,
        name,
        svc,
        s,
        m,
        null,
        "resource-helper-test",
        "test.instana.io",
        "444",
        0.5,
        512,
        1.0,
        1024,
        "instana/agent",
        "latest",
        null,
        null,
        null,
        null,
        null,
        null,
        "*");

    System.out.println(mapper.writeValueAsString(ds));

    assertThat("daemonset was created", null != client.apps().daemonSets().create(ds).getMetadata().getUid());

    //    for (; ; ) {
    //      Thread.sleep(5000);
    //    }
  }

}
