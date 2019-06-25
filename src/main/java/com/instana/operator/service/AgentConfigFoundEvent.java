package com.instana.operator.service;

import com.instana.operator.customresource.InstanaAgent;

public class AgentConfigFoundEvent {

  private final InstanaAgent config;

  public AgentConfigFoundEvent(InstanaAgent config) {
    this.config = config;
  }

  public InstanaAgent getConfig() {
    return config;
  }
}
