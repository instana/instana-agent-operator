package com.instana.operator.service;

import io.fabric8.kubernetes.api.model.ConfigMap;
import io.fabric8.kubernetes.api.model.ContainerBuilder;
import io.fabric8.kubernetes.api.model.Namespace;
import io.fabric8.kubernetes.api.model.Pod;
import io.fabric8.kubernetes.client.Config;
import io.fabric8.kubernetes.client.DefaultKubernetesClient;
import io.quarkus.test.junit.QuarkusTest;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import javax.inject.Inject;
import java.util.Collections;
import java.util.List;
import java.util.Optional;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.locks.LockSupport;
import java.util.function.Supplier;

import static com.instana.operator.util.ConfigUtils.createClientConfig;
import static com.instana.operator.util.OkHttpClientUtils.createHttpClient;
import static org.hamcrest.MatcherAssert.assertThat;

@QuarkusTest
class KubernetesResourceServiceIT {

  @Inject
  KubernetesResourceService kubernetesResourceService;
  Pod testPod;
  ConfigMap testConfigMap;

  DefaultKubernetesClient client;

  @BeforeEach
  void namespaceSetup() throws Exception {
    Config config = createClientConfig();
    client = new DefaultKubernetesClient(createHttpClient(config), config);
    Namespace ns = client.namespaces().withName("test").get();
    if (null == ns) {
      client.namespaces()
          .createNew()
          .withNewMetadata()
          .withName("test")
          .endMetadata()
          .done();
    }

    Supplier<List<Pod>> testPods = () -> client.pods()
        .inNamespace("test")
        .list()
        .getItems();

    testPods.get().forEach(p -> client.pods()
        .inNamespace("test")
        .withName(p.getMetadata().getName())
        .delete());
    client.configMaps()
        .inNamespace("test")
        .withName("test")
        .delete();

    int podCnt;
    do {
      podCnt = testPods.get().size();
      if (podCnt > 0) {
        LockSupport.parkNanos(TimeUnit.SECONDS.toNanos(1));
      }
    } while (podCnt > 0);

    this.testPod = client.pods().createNew()
        .withNewMetadata()
        .withNamespace("test")
        .withGenerateName("test-")
        .endMetadata()
        .withNewSpec()
        .withRestartPolicy("Never")
        .withContainers(new ContainerBuilder()
            .withName("hello-world")
            .withImage("hello-world")
            .build())
        .endSpec()
        .done();

    this.testConfigMap = client.configMaps().createNew()
        .withNewMetadata()
        .withNamespace("test")
        .withName("test")
        .endMetadata()
        .withData(Collections.singletonMap("hello", "world"))
        .done();
  }

  @Test
  void mustCachePodsInNamespace() throws InterruptedException {
    ResourceCache<Pod> pods = kubernetesResourceService.createResourceCache(
        "test",
        client -> client.watch("test", Pod.class));

    CountDownLatch latch = new CountDownLatch(1);
    pods.observe().subscribe(tup -> latch.countDown());

    latch.await(10, TimeUnit.SECONDS);

    Optional<Pod> p = pods.get(testPod.getMetadata().getName());
    assertThat("Pod available from cache", p.isPresent());
    assertThat("Pod UIDs match", testPod.getMetadata().getUid().equals(p.get().getMetadata().getUid()));
  }

  @Test
  void mustCacheConfigMap() throws InterruptedException {
    ResourceCache<ConfigMap> configMaps = kubernetesResourceService.createResourceCache(
        "test",
        client -> client.watch("test", ConfigMap.class));

    CountDownLatch latch = new CountDownLatch(1);
    configMaps.observe()
        .filter(cm -> "test".equals(cm.getName()))
        .subscribe(tup -> latch.countDown());

    latch.await(10, TimeUnit.SECONDS);

    Optional<ConfigMap> cm = configMaps.get("test");
    assertThat("ConfigMap available from cache", cm.isPresent());
    assertThat("ConfiMap UIDs match", testConfigMap.getMetadata().getUid().equals(cm.get().getMetadata().getUid()));
  }

}
