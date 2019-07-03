package com.instana.operator.cache;

public class FatalErrorHandler extends com.instana.operator.service.FatalErrorHandler {

  private boolean systemExitCalled = false;

  @Override
  public void systemExit(int status) {
    systemExitCalled = true;
  }

  boolean wasSystemExitCalled() {
    return systemExitCalled;
  }
}
