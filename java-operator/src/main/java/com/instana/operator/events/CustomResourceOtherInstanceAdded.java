/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc.
 */
package com.instana.operator.events;

import com.instana.operator.customresource.InstanaAgent;

public class CustomResourceOtherInstanceAdded {

  private final InstanaAgent currentInstance;
  private final InstanaAgent newInstance;

  public CustomResourceOtherInstanceAdded(InstanaAgent currentInstance, InstanaAgent newInstance) {
    this.currentInstance = currentInstance;
    this.newInstance = newInstance;
  }

  public InstanaAgent getCurrentInstance() {
    return currentInstance;
  }

  public InstanaAgent getNewInstance() {
    return newInstance;
  }
}
