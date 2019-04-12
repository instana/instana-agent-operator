package com.instana.operator.resource;

import static org.apache.commons.lang3.StringUtils.isBlank;

import javax.enterprise.context.ApplicationScoped;
import javax.inject.Inject;

import org.eclipse.microprofile.health.Health;
import org.eclipse.microprofile.health.HealthCheck;
import org.eclipse.microprofile.health.HealthCheckResponse;
import org.eclipse.microprofile.health.HealthCheckResponseBuilder;

import com.instana.operator.service.KubernetesResourceService;

@Health
@ApplicationScoped
public class HealthCheckResource implements HealthCheck {

  @Inject
  KubernetesResourceService clientService;

  @Override
  public HealthCheckResponse call() {
    // TODO: If we want to implement a health check resource we should inject it everywhere and make it fail as soon as there is an unrecoverable error or too many retries fail.
    // TODO: If we implement a readiness check it should be ready only if all onStartup() methods are done, because otherwise we get initialization errors after being ready.
    HealthCheckResponseBuilder b = HealthCheckResponse.named("instana-agent-operator");
    if (!isBlank(clientService.getKubernetesClient().getApiVersion())) {
      return b.up().build();
    } else {
      return b.down().build();
    }
  }

}
