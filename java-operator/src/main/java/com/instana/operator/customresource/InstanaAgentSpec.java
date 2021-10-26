/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc.
 */
package com.instana.operator.customresource;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.annotation.JsonDeserialize;
import com.fasterxml.jackson.dataformat.yaml.YAMLFactory;
import io.quarkus.runtime.annotations.RegisterForReflection;

import java.io.IOException;
import java.util.Collections;
import java.util.Map;

@JsonDeserialize
@JsonInclude(JsonInclude.Include.NON_DEFAULT)
@JsonIgnoreProperties(ignoreUnknown = true)
@RegisterForReflection
public class InstanaAgentSpec {

  static final String DEFAULT_AGENT_CLUSTER_ROLE_NAME = "instana-agent";
  static final String DEFAULT_AGENT_CLUSTER_ROLE_BINDING_NAME = "instana-agent";
  static final String DEFAULT_AGENT_SERVICE_ACCOUNT_NAME = "instana-agent";
  static final String DEFAULT_AGENT_SECRET_NAME = "instana-agent";
  static final String DEFAULT_AGENT_DAEMON_SET_NAME = "instana-agent";
  static final String DEFAULT_AGENT_CONFIG_MAP_NAME = "instana-agent";
  static final String DEFAULT_AGENT_RBAC_CREATE = "true";
  static final String DEFAULT_AGENT_CPU_REQ = "0.5";
  static final String DEFAULT_AGENT_CPU_LIMIT = "1.5";
  static final String DEFAULT_AGENT_MEM_REQ = "576";
  static final String DEFAULT_AGENT_MEM_LIMIT = "768";
  static final String DEFAULT_AGENT_IMAGE_PULLPOLICY = "Always";
  static final String DEFAULT_AGENT_OTEL_ACTIVE = "false";
  static final Integer DEFAULT_AGENT_OTEL_PORT = 55680;

  @JsonProperty("config.files")
  private Map<String, String> configFiles;
  @JsonProperty(value = "agent.zone.name")
  private String agentZoneName;
  @JsonProperty("agent.key")
  private String agentKey;
  @JsonProperty("agent.endpoint.host")
  private String agentEndpointHost;
  @JsonProperty("agent.endpoint.port")
  private int agentEndpointPort;
  @JsonProperty(value = "agent.clusterRoleName", defaultValue = DEFAULT_AGENT_CLUSTER_ROLE_NAME)
  private String agentClusterRoleName = DEFAULT_AGENT_CLUSTER_ROLE_NAME;
  @JsonProperty(value = "agent.clusterRoleBindingName", defaultValue = DEFAULT_AGENT_CLUSTER_ROLE_BINDING_NAME)
  private String agentClusterRoleBindingName = DEFAULT_AGENT_CLUSTER_ROLE_BINDING_NAME;
  @JsonProperty(value = "agent.serviceAccountName", defaultValue = DEFAULT_AGENT_SERVICE_ACCOUNT_NAME)
  private String agentServiceAccountName = DEFAULT_AGENT_SERVICE_ACCOUNT_NAME;
  @JsonProperty(value = "agent.secretName", defaultValue = DEFAULT_AGENT_SECRET_NAME)
  private String agentSecretName = DEFAULT_AGENT_SECRET_NAME;
  @JsonProperty(value = "agent.daemonSetName", defaultValue = DEFAULT_AGENT_DAEMON_SET_NAME)
  private String agentDaemonSetName = DEFAULT_AGENT_DAEMON_SET_NAME;
  @JsonProperty(value = "agent.configMapName", defaultValue = DEFAULT_AGENT_CONFIG_MAP_NAME)
  private String agentConfigMapName = DEFAULT_AGENT_CONFIG_MAP_NAME;
  @JsonProperty(value = "agent.rbac.create", defaultValue = DEFAULT_AGENT_RBAC_CREATE)
  private Boolean agentRbacCreate = Boolean.parseBoolean(DEFAULT_AGENT_RBAC_CREATE);
  @JsonProperty(value = "agent.image")
  private String agentImage;
  @JsonProperty(value = "agent.imagePullPolicy", defaultValue = DEFAULT_AGENT_IMAGE_PULLPOLICY)
  private String agentImagePullPolicy = DEFAULT_AGENT_IMAGE_PULLPOLICY;
  @JsonProperty(value = "agent.cpuReq", defaultValue = DEFAULT_AGENT_CPU_REQ)
  private Double agentCpuReq = Double.parseDouble(DEFAULT_AGENT_CPU_REQ);
  @JsonProperty(value = "agent.cpuLimit", defaultValue = DEFAULT_AGENT_CPU_LIMIT)
  private Double agentCpuLimit = Double.parseDouble(DEFAULT_AGENT_CPU_LIMIT);
  @JsonProperty(value = "agent.memReq", defaultValue = DEFAULT_AGENT_MEM_REQ)
  private Integer agentMemReq = Integer.parseInt(DEFAULT_AGENT_MEM_REQ);
  @JsonProperty(value = "agent.memLimit", defaultValue = DEFAULT_AGENT_MEM_LIMIT)
  private Integer agentMemLimit = Integer.parseInt(DEFAULT_AGENT_MEM_LIMIT);
  @JsonProperty(value = "agent.downloadKey")
  private String agentDownloadKey;
  @JsonProperty(value = "agent.host.repository")
  private String agentHostRepository;
  @JsonProperty(value = "opentelemetry.enabled", defaultValue = DEFAULT_AGENT_OTEL_ACTIVE)
  private Boolean agentOpenTelemetryEnabled = Boolean.parseBoolean(DEFAULT_AGENT_OTEL_ACTIVE);
  @JsonProperty(value = "cluster.name")
  private String clusterName;
  @JsonProperty(value = "agent.env")
  private Map<String, String> agentEnv;
  @JsonProperty(value = "agent.tls.secretName")
  private String agentTlsSecretName;
  @JsonProperty(value = "agent.tls.certificate")
  private String agentTlsCertificate;
  @JsonProperty(value = "agent.tls.key")
  private String agentTlsKey;

  public Map<String, String> getConfigFiles() {
    if (configFiles == null) {
      return Collections.emptyMap();
    } else {
      return configFiles;
    }
  }

  public void setConfigFiles(Map<String, String> configFiles) {
    this.configFiles = configFiles;
  }

  public String getAgentZoneName() {
    return agentZoneName;
  }

  public void setAgentZoneName(String agentZoneName) {
    this.agentZoneName = agentZoneName;
  }

  public String getAgentKey() {
    return agentKey;
  }

  public void setAgentKey(String agentKey) {
    this.agentKey = agentKey;
  }

  public String getAgentEndpointHost() {
    return agentEndpointHost;
  }

  public void setAgentEndpointHost(String agentEndpointHost) {
    this.agentEndpointHost = agentEndpointHost;
  }

  public int getAgentEndpointPort() {
    return agentEndpointPort;
  }

  public void setAgentEndpointPort(int agentEndpointPort) {
    this.agentEndpointPort = agentEndpointPort;
  }

  public String getAgentClusterRoleName() {
    return agentClusterRoleName;
  }

  public void setAgentClusterRoleName(String agentClusterRoleName) {
    this.agentClusterRoleName = agentClusterRoleName;
  }

  public String getAgentClusterRoleBindingName() {
    return agentClusterRoleBindingName;
  }

  public void setAgentClusterRoleBindingName(String agentClusterRoleBindingName) {
    this.agentClusterRoleBindingName = agentClusterRoleBindingName;
  }

  public String getAgentServiceAccountName() {
    return agentServiceAccountName;
  }

  public void setAgentServiceAccountName(String agentServiceAccountName) {
    this.agentServiceAccountName = agentServiceAccountName;
  }

  public String getAgentSecretName() {
    return agentSecretName;
  }

  public void setAgentSecretName(String agentSecretName) {
    this.agentSecretName = agentSecretName;
  }

  public String getAgentDaemonSetName() {
    return agentDaemonSetName;
  }

  public void setAgentDaemonSetName(String agentDaemonSetName) {
    this.agentDaemonSetName = agentDaemonSetName;
  }

  public String getAgentConfigMapName() {
    return agentConfigMapName;
  }

  public void setAgentConfigMapName(String agentConfigMapName) {
    this.agentConfigMapName = agentConfigMapName;
  }

  public boolean isAgentRbacCreate() {
    return agentRbacCreate;
  }

  public void setAgentRbacCreate(boolean agentRbacCreate) {
    this.agentRbacCreate = agentRbacCreate;
  }

  public String getAgentImage() {
    return agentImage;
  }

  public void setAgentImage(String agentImage) {
    this.agentImage = agentImage;
  }

  public Double getAgentCpuReq() {
    return agentCpuReq;
  }

  public void setAgentCpuReq(Double agentCpuReq) {
    this.agentCpuReq = agentCpuReq;
  }

  public Double getAgentCpuLimit() {
    return agentCpuLimit;
  }

  public void setAgentCpuLimit(Double agentCpuLimit) {
    this.agentCpuLimit = agentCpuLimit;
  }

  public Integer getAgentMemReq() {
    return agentMemReq;
  }

  public void setAgentMemReq(Integer agentMemReq) {
    this.agentMemReq = agentMemReq;
  }

  public Integer getAgentMemLimit() {
    return agentMemLimit;
  }

  public void setAgentMemLimit(Integer agentMemLimit) {
    this.agentMemLimit = agentMemLimit;
  }

  public String getAgentDownloadKey() {
    return agentDownloadKey;
  }

  public void setAgentDownloadKey(String agentDownloadKey) {
    this.agentDownloadKey = agentDownloadKey;
  }

  public String getAgentHostRepository() {
    return agentHostRepository;
  }

  public void setAgentHostRepository(String agentHostRepository) {
    this.agentHostRepository = agentHostRepository;
  }

  public String getClusterName() {
    return clusterName;
  }

  public String getAgentImagePullPolicy() { return agentImagePullPolicy; }

  public void setAgentImagePullPolicy(String imagePullPolicy) { this.agentImagePullPolicy = imagePullPolicy; }

  public Boolean getAgentOpenTelemetryEnabled() { return agentOpenTelemetryEnabled; }

  public void setAgentOpenTelemetryEnabled(Boolean agentOpenTelemetryEnabled) { this.agentOpenTelemetryEnabled = agentOpenTelemetryEnabled; }

  public Integer getAgentOtelPort() { return DEFAULT_AGENT_OTEL_PORT; }

  public Map<String, String> getAgentEnv() {
    if (agentEnv == null)
      return Collections.emptyMap();
    else
      return agentEnv;
  }

  public void setAgentEnv(Map<String, String> env) {
    agentEnv = env;
  }

  public String getAgentTlsSecretName() {
    return agentTlsSecretName;
  }

  public void setAgentTlsSecretName(String agentTlsSecretName) {
    this.agentTlsSecretName = agentTlsSecretName;
  }

  public String getAgentTlsCertificate() {
    return agentTlsCertificate;
  }

  public void setAgentTlsCertificate(String agentTlsCertificate) {
    this.agentTlsCertificate = agentTlsCertificate;
  }

  public String getAgentTlsKey() {
    return agentTlsKey;
  }

  public void setAgentTlsKey(String agentTlsKey) {
    this.agentTlsKey = agentTlsKey;
  }

  // We call equals() to check if the Spec was updated.
  // We serialize to YAML and compare the Strings, because this works even if somebody
  // adds a field and forgets to update the equals method.
  // Moreover, this ignores fields that are ignored by YAML serialization which is what we want.
  @Override
  public boolean equals(Object o) {
    try {
      if (this == o) {
        return true;
      }
      if (o == null || getClass() != o.getClass()) {
        return false;
      }
      InstanaAgentSpec that = (InstanaAgentSpec) o;
      ObjectMapper mapper = new ObjectMapper(new YAMLFactory());
      String thisString = mapper.writerWithDefaultPrettyPrinter().writeValueAsString(this);
      String thatString = mapper.writerWithDefaultPrettyPrinter().writeValueAsString(that);
      return thisString.equals(thatString);
    } catch (IOException e) {
      // I don't see how this can happen.
      throw new RuntimeException(e);
    }
  }
}
