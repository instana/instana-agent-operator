package com.instana.agent.kubernetes.operator;

import static com.instana.agent.kubernetes.operator.util.ConfigUtils.createKubernetesClientConfig;
import static com.instana.agent.kubernetes.operator.util.OkHttpClientUtils.createHttpClient;
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
import io.fabric8.kubernetes.api.model.rbac.KubernetesClusterRole;
import io.fabric8.kubernetes.api.model.rbac.KubernetesClusterRoleBinding;
import io.fabric8.kubernetes.client.Config;
import io.fabric8.openshift.client.DefaultOpenShiftClient;
import io.fabric8.openshift.client.OpenShiftConfig;

class ResourceHelperIT {

  YAMLMapper mapper;
  DefaultOpenShiftClient client;
  Namespace agentNS;
  String uuid;
  String name;

  @BeforeEach
  void setUp() throws Exception {
    Config config = createKubernetesClientConfig();
    client = new DefaultOpenShiftClient(createHttpClient(config), new OpenShiftConfig(config));
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
    ServiceAccount svc = ResourceHelper.createServiceAccount(uuid, name);
    System.out.println(mapper.writeValueAsString(svc));
    assertThat("serviceaccount was created", null != client.serviceAccounts().create(svc).getMetadata().getUid());

    // Secret
    Secret s = ResourceHelper.createAgentKeySecret(uuid, name, "Rlk1bmNxNE5UZmk2LWxtWVBCanJhQQo=");
    System.out.println(mapper.writeValueAsString(s));
    assertThat("secret was created", null != client.secrets().create(s).getMetadata().getUid());

    // ConfigMap
    ConfigMap m = ResourceHelper.createConfigurationConfigMap(uuid, name);
    System.out.println(mapper.writeValueAsString(m));
    assertThat("configmap was created", null != client.configMaps().create(m).getMetadata().getUid());

    // ClusterRole
    KubernetesClusterRole r = ResourceHelper.createAgentClusterRole(name);
    System.out.println(mapper.writeValueAsString(r));
    assertThat("clusterrole was created",
        null != client.rbac().kubernetesClusterRoles().createOrReplace(r).getMetadata().getUid());

    // ClusterRoleBinding
    KubernetesClusterRoleBinding rb = ResourceHelper.createAgentClusterRoleBinding(uuid, name);
    System.out.println(mapper.writeValueAsString(r));
    assertThat("clusterrolebinding was created",
        null != client.rbac().kubernetesClusterRoleBindings().createOrReplace(rb).getMetadata().getUid());

    // DaemonSet
    DaemonSet ds = ResourceHelper.createAgentDaemonSet(
        uuid,
        name,
        "resource-helper-test",
        "test-fullstack-0-us-west-2.instana.io",
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