package com.instana.operator;

import static com.instana.agent.kubernetes.operator.util.ConfigUtils.createKubernetesClientConfig;
import static com.instana.agent.kubernetes.operator.util.OkHttpClientUtils.createHttpClient;

import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.event.Observes;
import javax.enterprise.inject.Produces;
import javax.inject.Singleton;

import com.fasterxml.jackson.databind.ObjectMapper;

import io.fabric8.kubernetes.client.Config;
import io.fabric8.kubernetes.client.DefaultKubernetesClient;
import io.fabric8.kubernetes.client.NamespacedKubernetesClient;
import io.quarkus.runtime.ShutdownEvent;
import io.quarkus.runtime.StartupEvent;

@ApplicationScoped
public class InstanaOperator {

  @Produces
  @Singleton
  public ObjectMapper objectMapper() {
    return new ObjectMapper();
  }

  @Produces
  @Singleton
  public Config kubernetesClientConfig() throws Exception {
    return createKubernetesClientConfig();
  }

  @Produces
  @Singleton
  public NamespacedKubernetesClient namespacedKubernetesClient(Config config) {
    return new DefaultKubernetesClient(createHttpClient(config), config);
  }

  @Produces
  @Singleton
  public ScheduledExecutorService scheduledExecutorService() {
    return Executors.newScheduledThreadPool(Runtime.getRuntime().availableProcessors());
  }

  void startup(@Observes StartupEvent ev) {

  }

  void shutdown(@Observes ShutdownEvent ev) {

  }

}
