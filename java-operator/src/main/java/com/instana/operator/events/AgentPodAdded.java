/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc.
 */
package com.instana.operator.events;

import io.fabric8.kubernetes.api.model.Pod;

public class AgentPodAdded {

  private final Pod pod;

  public AgentPodAdded(Pod pod) {
    this.pod = pod;
  }

  public Pod getPod() {
    return pod;
  }
}
