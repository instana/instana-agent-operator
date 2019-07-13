package com.instana.operator.events;

import io.fabric8.kubernetes.api.model.Pod;

public class OperatorPodRunning {

  private final Pod operatorPod;

  public OperatorPodRunning(Pod operatorPod) {
    this.operatorPod = operatorPod;
  }

  public Pod getOperatorPod() {
    return operatorPod;
  }
}
