package com.instana.operator.customresource;

import io.fabric8.kubernetes.client.CustomResource;

public class ElectedLeader extends CustomResource {

  private ElectedLeaderSpec spec;

  public ElectedLeaderSpec getSpec() {
    return spec;
  }

  public void setSpec(ElectedLeaderSpec spec) {
    this.spec = spec;
  }

  @Override
  public String toString() {
    return "ElectedLeader{" +
        "apiVersion='" + getApiVersion() + '\'' +
        ", metadata=" + getMetadata() +
        ", spec=" + spec +
        '}';
  }

}
