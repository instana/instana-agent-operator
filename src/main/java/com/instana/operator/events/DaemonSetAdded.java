package com.instana.operator.events;

import io.fabric8.kubernetes.api.model.apps.DaemonSet;

public class DaemonSetAdded {

  private final DaemonSet daemonSet;

  public DaemonSetAdded(DaemonSet daemonSet) {
    this.daemonSet = daemonSet;
  }

  public DaemonSet getDaemonSet() {
    return daemonSet;
  }
}
