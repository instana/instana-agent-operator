package com.instana.operator;

public class GlobalErrorEvent {
  private final Throwable cause;

  public GlobalErrorEvent(Throwable cause) {
    this.cause = cause;
  }

  public Throwable getCause() {
    return cause;
  }
}
