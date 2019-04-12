package com.instana.operator.service;

import com.instana.operator.GlobalErrorEvent;
import com.instana.operator.config.InstanaConfig;
import io.fabric8.kubernetes.api.model.ConfigMap;
import io.fabric8.kubernetes.client.NamespacedKubernetesClient;
import io.quarkus.runtime.StartupEvent;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.event.Event;
import javax.enterprise.event.Observes;
import javax.inject.Inject;
import java.util.concurrent.CompletableFuture;

@ApplicationScoped
public class InstanaConfigService {

  private static final String INSTANA_AGENT_CONFIG_NAME = "config";

  @Inject
  NamespacedKubernetesClient kubernetesClient;
  @Inject
  OperatorNamespaceService namespaceService;
  @Inject
  Event<GlobalErrorEvent> globalErrorEvent;

  private final CompletableFuture<InstanaConfig> config = new CompletableFuture<>();

  void onStartup(@Observes StartupEvent _ev) {
    ConfigMap operatorConfigMap = kubernetesClient.configMaps()
        .inNamespace(namespaceService.getNamespace())
        .withName(INSTANA_AGENT_CONFIG_NAME)
        .get();
    if (null == operatorConfigMap) {
      globalErrorEvent.fire(new GlobalErrorEvent(new IllegalStateException(
          "Operator ConfigMap named " + INSTANA_AGENT_CONFIG_NAME + " not found in namespace "
              + namespaceService.getNamespace())));
      return;
    }
    try {
      config.complete(new InstanaConfig(operatorConfigMap.getData()));
    } catch (IllegalArgumentException e) {
      globalErrorEvent.fire(new GlobalErrorEvent(e));
    }
  }

  public InstanaConfig getConfig() {
    try {
      return config.get();
    } catch (Exception e) {
      globalErrorEvent.fire(new GlobalErrorEvent(e));
      return null; // Will not happen, because firing the event above will abort the operator.
    }
  }
}
