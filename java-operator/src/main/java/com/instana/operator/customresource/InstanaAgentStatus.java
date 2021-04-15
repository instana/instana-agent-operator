/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc.
 */
package com.instana.operator.customresource;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.databind.annotation.JsonDeserialize;
import io.quarkus.runtime.annotations.RegisterForReflection;

@JsonDeserialize
@JsonIgnoreProperties(ignoreUnknown = true)
@RegisterForReflection
public class InstanaAgentStatus {

  @JsonProperty("serviceaccount")
  private ResourceInfo serviceAccount;
  @JsonProperty("clusterrole")
  private ResourceInfo clusterRole;
  @JsonProperty("clusterrolebinding")
  private ResourceInfo clusterRoleBinding;
  @JsonProperty("secret")
  private ResourceInfo secret;
  @JsonProperty("configmap")
  private ResourceInfo configMap;
  @JsonProperty("daemonset")
  private ResourceInfo daemonSet;
  @JsonProperty("leading.agent.pod")
  private ResourceInfo leadingAgentPod;

  public ResourceInfo getServiceAccount() {
    return serviceAccount;
  }

  public void setServiceAccount(ResourceInfo serviceAccount) {
    this.serviceAccount = serviceAccount;
  }

  public ResourceInfo getClusterRole() {
    return clusterRole;
  }

  public void setClusterRole(ResourceInfo clusterRole) {
    this.clusterRole = clusterRole;
  }

  public ResourceInfo getClusterRoleBinding() {
    return clusterRoleBinding;
  }

  public void setClusterRoleBinding(ResourceInfo clusterRoleBinding) {
    this.clusterRoleBinding = clusterRoleBinding;
  }

  public ResourceInfo getSecret() {
    return secret;
  }

  public void setSecret(ResourceInfo secret) {
    this.secret = secret;
  }

  public ResourceInfo getConfigMap() {
    return configMap;
  }

  public void setConfigMap(ResourceInfo configMap) {
    this.configMap = configMap;
  }

  public ResourceInfo getDaemonSet() {
    return daemonSet;
  }

  public void setDaemonSet(ResourceInfo daemonSet) {
    this.daemonSet = daemonSet;
  }

  public ResourceInfo getLeadingAgentPod() {
    return leadingAgentPod;
  }

  public void setLeadingAgentPod(ResourceInfo leadingAgentPod) {
    this.leadingAgentPod = leadingAgentPod;
  }
}
