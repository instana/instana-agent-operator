package com.instana.operator.customresource;

import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.databind.annotation.JsonDeserialize;

@JsonDeserialize
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
