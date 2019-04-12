package com.instana.operator.service;

import static org.hamcrest.MatcherAssert.assertThat;

import java.util.Collections;
import java.util.List;
import java.util.Optional;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.locks.LockSupport;
import java.util.function.Supplier;

import javax.inject.Inject;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import io.fabric8.kubernetes.api.model.ConfigMap;
import io.fabric8.kubernetes.api.model.ContainerBuilder;
import io.fabric8.kubernetes.api.model.Namespace;
import io.fabric8.kubernetes.api.model.Pod;
import io.quarkus.test.junit.QuarkusTest;

@QuarkusTest
class KubernetesResourceServiceIT {

  static final Logger LOGGER = LoggerFactory.getLogger(KubernetesResourceServiceIT.class);

  @Inject
  KubernetesResourceService kubernetesResourceService;
  Pod testPod;
  ConfigMap testConfigMap;

  @BeforeEach
  void namespaceSetup() {
    Namespace ns = kubernetesResourceService.getKubernetesClient().namespaces().withName("test").get();
    if (null == ns) {
      kubernetesResourceService.getKubernetesClient().namespaces()
          .createNew()
          .withNewMetadata()
          .withName("test")
          .endMetadata()
          .done();
    }

    Supplier<List<Pod>> testPods = () -> kubernetesResourceService.getKubernetesClient().pods()
        .inNamespace("test")
        .list()
        .getItems();

    testPods.get().forEach(p -> kubernetesResourceService.getKubernetesClient().pods()
        .inNamespace("test")
        .withName(p.getMetadata().getName())
        .delete());
    kubernetesResourceService.getKubernetesClient().configMaps()
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

    this.testPod = kubernetesResourceService.getKubernetesClient().pods().createNew()
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

    this.testConfigMap = kubernetesResourceService.getKubernetesClient().configMaps().createNew()
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
        client -> client.pods().inNamespace("test"));

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
        client -> client.configMaps().inNamespace("test"));

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
