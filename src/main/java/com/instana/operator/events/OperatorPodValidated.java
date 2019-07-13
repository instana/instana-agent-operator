package com.instana.operator.events;

import io.fabric8.kubernetes.api.model.Pod;

public class OperatorPodValidated {

  private final Pod operatorPod;

  public OperatorPodValidated(Pod operatorPod) {
    this.operatorPod = operatorPod;
  }

  public Pod getOperatorPod() {
    return operatorPod;
  }
}
