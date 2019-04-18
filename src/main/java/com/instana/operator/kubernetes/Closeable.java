package com.instana.operator.kubernetes;

public interface Closeable extends AutoCloseable {

  /**
   * No checked Exception.
   */
  @Override
  void close();
}
