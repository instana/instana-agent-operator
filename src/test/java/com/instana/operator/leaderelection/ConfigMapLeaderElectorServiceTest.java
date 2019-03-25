package com.instana.operator.leaderelection;

import javax.inject.Inject;

import io.fabric8.kubernetes.client.NamespacedKubernetesClient;
import io.quarkus.test.junit.QuarkusTest;

@QuarkusTest
class ConfigMapLeaderElectorServiceTest {

  @Inject
  NamespacedKubernetesClient kubernetesClient;

}