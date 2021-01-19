/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc.
 */
package com.instana.operator.cache;

public class FatalErrorHandler extends com.instana.operator.FatalErrorHandler {

  private boolean systemExitCalled = false;

  @Override
  public void systemExit(int status) {
    systemExitCalled = true;
  }

  boolean wasSystemExitCalled() {
    return systemExitCalled;
  }
}
