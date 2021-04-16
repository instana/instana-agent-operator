/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc.
 */
package com.instana.operator.events;

import com.instana.operator.customresource.InstanaAgent;

public class CustomResourceAdded {

  private final InstanaAgent instanaAgentResource;

  public CustomResourceAdded(InstanaAgent instanaAgentResource) {
    this.instanaAgentResource = instanaAgentResource;
  }

  public InstanaAgent getInstanaAgentResource() {
    return instanaAgentResource;
  }
}
