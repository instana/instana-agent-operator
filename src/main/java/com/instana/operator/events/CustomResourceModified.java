/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc.
 */
package com.instana.operator.events;

import com.instana.operator.customresource.InstanaAgent;

public class CustomResourceModified {

  private final InstanaAgent current;
  private final InstanaAgent next;

  public CustomResourceModified(InstanaAgent current, InstanaAgent next) {
    this.current = current;
    this.next = next;
  }

  public InstanaAgent getCurrent() {
    return current;
  }

  public InstanaAgent getNext() {
    return next;
  }
}
