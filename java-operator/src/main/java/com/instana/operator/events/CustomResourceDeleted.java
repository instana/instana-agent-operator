/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc.
 */
package com.instana.operator.events;

import com.instana.operator.customresource.InstanaAgent;

public class CustomResourceDeleted {

  private final InstanaAgent instanaAgentResource;

  public CustomResourceDeleted(InstanaAgent instanaAgentResource) {
    this.instanaAgentResource = instanaAgentResource;
  }

  public InstanaAgent getInstanaAgentResource() {
    return instanaAgentResource;
  }
}
