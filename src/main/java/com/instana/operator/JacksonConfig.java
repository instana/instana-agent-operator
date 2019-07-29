package com.instana.operator;

import com.fasterxml.jackson.databind.DeserializationFeature;
import io.fabric8.kubernetes.client.utils.Serialization;
import io.quarkus.runtime.StartupEvent;

import javax.annotation.Priority;
import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.event.Observes;

import static javax.interceptor.Interceptor.Priority.APPLICATION;

@ApplicationScoped
public class JacksonConfig {

  // This must happen before interacting with the API server, therefore we set priority APPLICATION - 1.
  public void onStartup(@Observes @Priority(APPLICATION - 1) StartupEvent _ev) {
    Serialization.jsonMapper().configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
    Serialization.yamlMapper().configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
  }
}
