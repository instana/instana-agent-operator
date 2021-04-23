/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc.
 */
package com.instana.operator.cache;

import io.fabric8.kubernetes.api.model.Pod;
import io.fabric8.kubernetes.api.model.PodList;
import io.reactivex.disposables.Disposable;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.util.Optional;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.concurrent.atomic.AtomicReference;

class CacheTest {

  private CacheService cacheService;
  private FatalErrorHandler errorHandler;
  private Cache<Pod, PodList> podCache;

  @BeforeEach
  void setUp() {
    errorHandler = new FatalErrorHandler();
    errorHandler.onStartup(null);
    cacheService = new CacheService();
    cacheService.fatalErrorHandler = errorHandler;
    podCache = cacheService.newCache(Pod.class, PodList.class);
  }

  @AfterEach
  void tearDown() throws Exception {
    cacheService.terminate(5, TimeUnit.SECONDS);
  }

  @Test
  void testGoodCase() throws Exception {

    final AtomicReference<Optional<Pod>> pod1 = new AtomicReference<>();
    final AtomicReference<Optional<Pod>> pod2 = new AtomicReference<>();

    try (KubernetesSimulator simulator = new KubernetesSimulator()) {
      simulator.simulatePodAdded("uid1", "pod1", 123);
      podCache.listThenWatch(simulator).subscribe(
          event -> {
            switch (event.getUid()) {
            case "uid1":
              pod1.set(podCache.get(event.getUid()));
              break;
            case "uid2":
              pod2.set(podCache.get(event.getUid()));
              break;
            default:
              Assertions.fail("unexpected uid " + event.getUid());
            }
          }
      );
      simulator.simulatePodModified("uid1", 125);
      simulator.simulatePodModified("uid1", 139);
      simulator.simulatePodAdded("uid2", "pod2", 789);
      simulator.simulatePodModified("uid2", 789); // similar to the previous version so ignored
      simulator.simulatePodDeleted("uid1");
    }
    cacheService.terminate(5, TimeUnit.SECONDS);
    Assertions.assertFalse(pod1.get().isPresent());
    Assertions.assertTrue(pod2.get().isPresent());
    Assertions.assertEquals("789", pod2.get().get().getMetadata().getResourceVersion());
    Assertions.assertFalse(errorHandler.wasSystemExitCalled());
  }

  @Test
  void testExceptionInEventProcessor() throws Exception {
    try (KubernetesSimulator simulator = new KubernetesSimulator()) {
      podCache.listThenWatch(simulator).subscribe(
          event -> {throw new RuntimeException("this should trigger System.exit(-1)");}
      );
      simulator.simulatePodAdded("uid1", "pod1", 1);
    }
    cacheService.terminate(5, TimeUnit.SECONDS);
    Assertions.assertTrue(errorHandler.wasSystemExitCalled());
  }

  @Test
  void testWatchErrorsAreHandled() throws Exception {
    AtomicBoolean errorHandlerCalled = new AtomicBoolean(false);
    try (KubernetesSimulator simulator = new KubernetesSimulator()) {
      podCache.listThenWatch(simulator).subscribe(
          event -> {},
          ex -> errorHandlerCalled.set(true)
      );
      simulator.simulateError();
    }
    cacheService.terminate(5, TimeUnit.SECONDS);
    Assertions.assertFalse(errorHandlerCalled.get());
  }

  @Test
  void testDispose() throws Exception {
    final AtomicInteger numberEventsReceived = new AtomicInteger(0);
    try (KubernetesSimulator simulator = new KubernetesSimulator()) {
      simulator.simulatePodAdded("test", "test", 1);
      Disposable watch = podCache.listThenWatch(simulator).subscribe(
          event -> numberEventsReceived.incrementAndGet()
      );
      Thread.sleep(100); // receive first event
      watch.dispose();
      simulator.simulatePodModified("test", 2);
      simulator.simulatePodModified("test", 3);
      Assertions.assertTrue(simulator.isWatchCloseCalled());
      Assertions.assertEquals(1, numberEventsReceived.get()); // only initial ADDED event.
    }
    cacheService.terminate(5, TimeUnit.SECONDS);
  }
}
