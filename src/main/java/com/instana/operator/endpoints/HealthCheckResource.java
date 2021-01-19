/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc.
 */
package com.instana.operator.endpoints;

import org.eclipse.microprofile.health.Health;
import org.eclipse.microprofile.health.HealthCheck;
import org.eclipse.microprofile.health.HealthCheckResponse;
import org.eclipse.microprofile.health.HealthCheckResponseBuilder;

import javax.enterprise.context.ApplicationScoped;

@Health
@ApplicationScoped
public class HealthCheckResource implements HealthCheck {

  @Override
  public HealthCheckResponse call() {
    // TODO: This is not too useful at the moment.
    HealthCheckResponseBuilder b = HealthCheckResponse.named("instana-agent-operator");
      return b.up().build();
  }
}
