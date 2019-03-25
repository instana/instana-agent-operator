package com.instana.operator.resource;

import static org.apache.commons.lang3.StringUtils.isBlank;

import javax.enterprise.context.ApplicationScoped;
import javax.inject.Inject;

import org.eclipse.microprofile.health.Health;
import org.eclipse.microprofile.health.HealthCheck;
import org.eclipse.microprofile.health.HealthCheckResponse;
import org.eclipse.microprofile.health.HealthCheckResponseBuilder;

import io.fabric8.kubernetes.client.NamespacedKubernetesClient;

@Health
@ApplicationScoped
public class HealthCheckResource implements HealthCheck {

  @Inject
  NamespacedKubernetesClient kubernetesClient;

  @Override
  public HealthCheckResponse call() {
    HealthCheckResponseBuilder b = HealthCheckResponse.named("instana-operator");
    if (!isBlank(kubernetesClient.getApiVersion())) {
      return b.up().build();
    } else {
      return b.down().build();
    }
  }

}
