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

class ResourceWatchTest {

  private ResourceService resourceService;
  private FatalErrorHandler errorHandler;
  private ResourceWatch<Pod, PodList> podResourceWatch;

  @BeforeEach
  void setUp() {
    errorHandler = new FatalErrorHandler();
    errorHandler.onStartup(null);
    resourceService = new ResourceService();
    resourceService.fatalErrorHandler = errorHandler;
    podResourceWatch = resourceService.newResourceWatch(PodList.class);
  }

  @AfterEach
  void tearDown() throws Exception {
    resourceService.terminate(5, TimeUnit.SECONDS);
  }

  @Test
  void testGoodCase() throws Exception {

    final AtomicReference<Optional<Pod>> pod1 = new AtomicReference<>();
    final AtomicReference<Optional<Pod>> pod2 = new AtomicReference<>();

    try (KubernetesSimulator simulator = new KubernetesSimulator()) {
      simulator.simulatePodAdded("uid1", "pod1", 123);
      podResourceWatch.listThenWatch(simulator).subscribe(
          event -> {
            switch (event.getUid()) {
            case "uid1":
              pod1.set(podResourceWatch.get(event.getUid()));
              break;
            case "uid2":
              pod2.set(podResourceWatch.get(event.getUid()));
              break;
            default:
              Assertions.fail("unexpected uid " + event.getUid());
            }
          }
                                                         );
      simulator.simulatePodModified("uid1", 125);
      simulator.simulatePodModified("uid1", 139);
      simulator.simulatePodAdded("uid2", "pod2", 789);
      simulator.simulatePodModified("uid2", 633); // less than the previous version for pod2
      simulator.simulatePodDeleted("uid1");
    }
    resourceService.terminate(5, TimeUnit.SECONDS);
    Assertions.assertFalse(pod1.get().isPresent());
    Assertions.assertTrue(pod2.get().isPresent());
    Assertions.assertEquals("789", pod2.get().get().getMetadata().getResourceVersion());
    Assertions.assertFalse(errorHandler.wasSystemExitCalled());
  }

  @Test
  void testExceptionInEventProcessor() throws Exception {
    try (KubernetesSimulator simulator = new KubernetesSimulator()) {
      podResourceWatch.listThenWatch(simulator).subscribe(
          event -> {throw new RuntimeException("this should trigger System.exit(-1)");}
                                                         );
      simulator.simulatePodAdded("uid1", "pod1", 1);
    }
    resourceService.terminate(5, TimeUnit.SECONDS);
    Assertions.assertTrue(errorHandler.wasSystemExitCalled());
  }

  @Test
  void testErrorHandlerCalled() throws Exception {
    AtomicBoolean errorHandlerCalled = new AtomicBoolean(false);
    try (KubernetesSimulator simulator = new KubernetesSimulator()) {
      podResourceWatch.listThenWatch(simulator).subscribe(
          event -> {},
          ex -> {errorHandlerCalled.set(true);}
                                                         );
      simulator.simulateError();
    }
    resourceService.terminate(5, TimeUnit.SECONDS);
    Assertions.assertTrue(errorHandlerCalled.get());
    Assertions.assertTrue(errorHandler.wasSystemExitCalled());
  }

  @Test
  void testExceptionInErrorHandler() throws Exception {
    try (KubernetesSimulator simulator = new KubernetesSimulator()) {
      podResourceWatch.listThenWatch(simulator).subscribe(
          event -> {},
          ex -> {throw new RuntimeException("test");}
                                                         );
      simulator.simulateError();
    }
    resourceService.terminate(5, TimeUnit.SECONDS);
    Assertions.assertTrue(errorHandler.wasSystemExitCalled());
  }

  @Test
  void testDispose() throws Exception {
    final AtomicInteger numberEventsReceived = new AtomicInteger(0);
    try (KubernetesSimulator simulator = new KubernetesSimulator()) {
      simulator.simulatePodAdded("test", "test", 1);
      Disposable watch = podResourceWatch.listThenWatch(simulator).subscribe(
          event -> {
            numberEventsReceived.incrementAndGet();
          }
                                                                            );
      Thread.sleep(100); // receive first event
      watch.dispose();
      simulator.simulatePodModified("test", 2);
      simulator.simulatePodModified("test", 3);
      Assertions.assertTrue(simulator.isWatchCloseCalled());
      Assertions.assertEquals(1, numberEventsReceived.get()); // only initial ADDED event.
    }
    resourceService.terminate(5, TimeUnit.SECONDS);
  }
}
