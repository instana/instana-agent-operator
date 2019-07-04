package com.instana.operator.customresource;

import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.databind.annotation.JsonDeserialize;
import io.quarkus.runtime.annotations.RegisterForReflection;

@JsonDeserialize
@RegisterForReflection
public class InstanaAgentStatus {

  @JsonProperty
  private boolean agentDeployed;

  public InstanaAgentStatus() {
    this.agentDeployed = false;
  }

  public boolean isAgentDeployed() {
    return agentDeployed;
  }

  public void setAgentDeployed(boolean agentDeployed) {
    this.agentDeployed = agentDeployed;
  }
}
