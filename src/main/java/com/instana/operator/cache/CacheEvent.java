package com.instana.operator.cache;

import io.fabric8.kubernetes.client.Watcher;

public class CacheEvent {

  private final Watcher.Action action;
  private final String uid;

  CacheEvent(Watcher.Action action, String uid) {
    this.action = action;
    this.uid = uid;
  }

  public Watcher.Action getAction() {
    return action;
  }

  public String getUid() {
    return uid;
  }
}
