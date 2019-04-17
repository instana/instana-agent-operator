package com.instana.operator.service;

import javax.annotation.Priority;
import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.inject.Alternative;

import io.fabric8.kubernetes.api.model.Pod;
import io.fabric8.kubernetes.client.DefaultKubernetesClient;
import io.fabric8.kubernetes.client.NamespacedKubernetesClient;

@Alternative
@Priority(1)
@ApplicationScoped
public class TestNamespaceService extends OperatorNamespaceService {

  private String name;
  private String namespace;

  public TestNamespaceService() {
    NamespacedKubernetesClient client = new DefaultKubernetesClient();
    Pod operatorPod = client.pods().inNamespace("instana-agent").list().getItems().stream()
        .filter(p -> p.getMetadata().getName().contains("instana-agent-operator"))
        .findFirst()
        .orElse(null);
    name = operatorPod.getMetadata().getName();
    namespace = operatorPod.getMetadata().getNamespace();
  }

  @Override
  synchronized String getOperatorPodName() {
    return name;
  }

  @Override
  public synchronized String getNamespace() {
    return namespace;
  }

}
