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
public class InstanaAgentConfigFiles {

  @JsonProperty("configuration.yaml")
  private String configurationYaml;

  public String getConfigurationYaml() {
    return configurationYaml;
  }

  public void setConfigurationYaml(String configurationYaml) {
    this.configurationYaml = configurationYaml;
  }
}
