package com.instana.operator.customresource;

import io.fabric8.kubernetes.client.CustomResource;

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
