package com.instana.operator;

public class GlobalErrorEvent {

  private final String msg;
  private final Throwable cause;

  public GlobalErrorEvent(String msg) {
    this(msg, null);
  }

  public GlobalErrorEvent(String msg, Throwable cause) {
    this.msg = msg;
    this.cause = cause;
  }

  public GlobalErrorEvent(Throwable cause) {
    this(cause != null ? cause.getMessage() : null, cause);
  }

  public String getMessage() {
    return msg;
  }

  public Throwable getCause() {
    return cause;
  }
}
