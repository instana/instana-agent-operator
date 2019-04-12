package com.instana.operator.config;

import java.util.Map;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.event.Event;
import javax.enterprise.event.Observes;
import javax.inject.Inject;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import com.instana.operator.GlobalErrorEvent;
import com.instana.operator.service.OperatorNamespaceService;

import io.fabric8.kubernetes.api.model.ConfigMap;
import io.fabric8.kubernetes.client.NamespacedKubernetesClient;
import io.quarkus.runtime.StartupEvent;

@ApplicationScoped
public class InstanaConfig {

  private static final Logger LOGGER = LoggerFactory.getLogger(InstanaConfig.class);

  private static final String INSTANA_AGENT_CONFIG_NAME = "config";

  @Inject
  NamespacedKubernetesClient kubernetesClient;
  @Inject
  OperatorNamespaceService namespaceService;
  @Inject
  Event<GlobalErrorEvent> globalErrorEvent;

  private String clusterRoleName;
  private String clusterRoleBindingName;
  private String serviceAccountName;
  private String secretName;
  private String configMapName;
  private String daemonSetName;

  private boolean rbacCreate;

  private String zoneName;
  private String agentKey;
  private String agentImageName;
  private String agentImageTag;
  private String endpoint;
  private String endpointPort;

  private double agentCpuReq;
  private double agentCpuLimit;
  private int agentMemReq;
  private int agentMemLimit;

  private String agentProxyHost;
  private String agentProxyPort;
  private String agentProxyProtocol;
  private String agentProxyUser;
  private String agentProxyPasswd;
  private String agentProxyUseDNS;
  private String agentHttpListen;

  void onStartup(@Observes StartupEvent _ev) {
    ConfigMap operatorConfigMap = kubernetesClient.configMaps()
        .inNamespace(namespaceService.getNamespace())
        .withName(INSTANA_AGENT_CONFIG_NAME)
        .get();
    if (null == operatorConfigMap) {
      globalErrorEvent.fire(new GlobalErrorEvent(new IllegalStateException(
          "Operator ConfigMap named " + INSTANA_AGENT_CONFIG_NAME + " not found in namespace "
              + namespaceService.getNamespace())));
    }
    Map<String, String> data = operatorConfigMap.getData();

    // Whether to create the RBAC resources or not, as the user may want to manage themselves.
    rbacCreate = Boolean.valueOf(data.getOrDefault("agent.rbac.create", "true"));
    // Names of the various Agent-related resources.
    serviceAccountName = data.getOrDefault("agent.serviceAccountName", "instana-agent");
    clusterRoleName = data.getOrDefault("agent.clusterRoleName", "instana-agent");
    clusterRoleBindingName = data.getOrDefault("agent.clusterRoleBindingName", "instana-agent");
    secretName = data.getOrDefault("agent.secretName", "instana-agent");
    configMapName = data.getOrDefault("agent.configMapName", "agent-config");
    daemonSetName = data.getOrDefault("agent.daemonSetName", "instana-agent");

    // Zone name is required to identify this cluster on the infra map.
    zoneName = data.getOrDefault("zone.name", System.getenv("INSTANA_ZONE"));
    if (null == zoneName) {
      LOGGER.error("'zone.name' is required to be set but was not set in the ConfigMap "
          + "nor was the INSTANA_ZONE environment variable present in the Operator's environment.");
      // Force a restart of the container if config is missing in order to pick up the change on restart.
      System.exit(1);
    }

    // Agent key is required or the agent won't start.
    agentKey = data.getOrDefault("agent.key", System.getenv("INSTANA_AGENT_KEY"));
    if (null == agentKey) {
      LOGGER.error("'agent.key' is required to be set but was not set in the ConfigMap "
          + "nor was the INSTANA_AGENT_KEY environment variable present in the Operator's environment.");
      // Force a restart of the container if config is missing in order to pick up the change on restart.
      System.exit(1);
    }

    agentImageName = data.getOrDefault("agent.image", "instana/agent");
    agentImageTag = data.getOrDefault("agent.imageTag", "latest");

    endpoint = data.getOrDefault("agent.endpoint", "saas-us-west-2.instana.io");
    endpointPort = data.getOrDefault("agent.endpoint.port", "443");

    agentCpuReq = Float.parseFloat(data.getOrDefault("agent.cpuReq", "0.5"));
    agentCpuLimit = Float.parseFloat(data.getOrDefault("agent.cpuLimit", "1.5"));
    agentMemReq = Integer.parseInt(data.getOrDefault("agent.memReq", "512"));
    agentMemLimit = Integer.parseInt(data.getOrDefault("agent.memLimit", "512"));

    agentProxyHost = data.get("agent.proxy.host");
    agentProxyPort = data.get("agent.proxy.port");
    agentProxyProtocol = data.get("agent.proxy.protocol");
    agentProxyUser = data.get("agent.proxy.user");
    agentProxyPasswd = data.get("agent.proxy.password");
    agentProxyUseDNS = data.get("agent.proxy.use.dns");
    agentHttpListen = data.getOrDefault("agent.http.listen", "*");
  }

  public String getClusterRoleName() {
    return clusterRoleName;
  }

  public String getClusterRoleBindingName() {
    return clusterRoleBindingName;
  }

  public String getServiceAccountName() {
    return serviceAccountName;
  }

  public String getSecretName() {
    return secretName;
  }

  public String getConfigMapName() {
    return configMapName;
  }

  public String getDaemonSetName() {
    return daemonSetName;
  }

  public boolean isRbacCreate() {
    return rbacCreate;
  }

  public String getZoneName() {
    return zoneName;
  }

  public String getAgentKey() {
    return agentKey;
  }

  public String getAgentImageName() {
    return agentImageName;
  }

  public String getAgentImageTag() {
    return agentImageTag;
  }

  public String getEndpoint() {
    return endpoint;
  }

  public String getEndpointPort() {
    return endpointPort;
  }

  public double getAgentCpuReq() {
    return agentCpuReq;
  }

  public double getAgentCpuLimit() {
    return agentCpuLimit;
  }

  public int getAgentMemReq() {
    return agentMemReq;
  }

  public int getAgentMemLimit() {
    return agentMemLimit;
  }

  public String getAgentProxyHost() {
    return agentProxyHost;
  }

  public String getAgentProxyPort() {
    return agentProxyPort;
  }

  public String getAgentProxyProtocol() {
    return agentProxyProtocol;
  }

  public String getAgentProxyUser() {
    return agentProxyUser;
  }

  public String getAgentProxyPasswd() {
    return agentProxyPasswd;
  }

  public String getAgentProxyUseDNS() {
    return agentProxyUseDNS;
  }

  public String getAgentHttpListen() {
    return agentHttpListen;
  }

}
