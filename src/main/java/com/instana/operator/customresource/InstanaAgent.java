package com.instana.operator.customresource;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.databind.annotation.JsonDeserialize;
import io.fabric8.kubernetes.client.CustomResource;
import io.quarkus.runtime.annotations.RegisterForReflection;

@JsonDeserialize
@JsonIgnoreProperties(ignoreUnknown = true)
@RegisterForReflection
public class InstanaAgent extends CustomResource {

  private InstanaAgentSpec spec;
  private InstanaAgentStatus status;

  public InstanaAgentSpec getSpec() {
    return spec;
  }

  public void setSpec(InstanaAgentSpec spec) {
    this.spec = spec;
  }

  public InstanaAgentStatus getStatus() {
    return status;
  }

  public void setStatus(InstanaAgentStatus status) {
    this.status = status;
  }
}
