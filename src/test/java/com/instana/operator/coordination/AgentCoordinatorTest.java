/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc.
 */
package com.instana.operator.coordination;

import com.instana.operator.events.AgentPodAdded;
import com.instana.operator.events.AgentPodDeleted;
import io.fabric8.kubernetes.api.model.Pod;
import io.fabric8.kubernetes.api.model.PodBuilder;
import org.hamcrest.Description;
import org.hamcrest.Matcher;
import org.hamcrest.TypeSafeMatcher;
import org.junit.jupiter.api.Test;
import org.mockito.Mockito;

import java.io.IOException;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collections;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Random;
import java.util.Set;
import java.util.UUID;
import java.util.concurrent.Delayed;
import java.util.concurrent.FutureTask;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.ScheduledFuture;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicReference;

import static java.util.Collections.singleton;
import static org.hamcrest.MatcherAssert.assertThat;
import static org.hamcrest.Matchers.containsInAnyOrder;
import static org.hamcrest.Matchers.equalTo;
import static org.hamcrest.Matchers.hasEntry;
import static org.hamcrest.Matchers.not;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyLong;
import static org.mockito.Mockito.atLeast;
import static org.mockito.Mockito.doAnswer;
import static org.mockito.Mockito.doThrow;
import static org.mockito.Mockito.eq;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.times;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.verifyNoMoreInteractions;
import static org.mockito.Mockito.when;

class AgentCoordinatorTest {
  private Map<Pod, CoordinationRecord> podStates;
  private PodCoordinationIO podCoordinationIO;

  @Test
  void mustAssignLeaderForPodBasedOnRequestedResources() throws IOException {
    setupMockPodCoordinationIO();

    MockExecutor mockExecutor = new MockExecutor();
    AgentCoordinator coordinator = new AgentCoordinator(
        podCoordinationIO, mock(Random.class), mockExecutor.getExecutor());

    Pod[] pods = new Pod[]{createPod(), createPod(), createPod()};

    setPodRequests(pods[0], "test-resource", "another-resource");
    setPodRequests(pods[1], "test-resource");
    setPodRequests(pods[2], "test-resource", "another-resource");

    coordinator.onAgentPodAdded(new AgentPodAdded(pods[0]));
    coordinator.onAgentPodAdded(new AgentPodAdded(pods[1]));
    coordinator.onAgentPodAdded(new AgentPodAdded(pods[2]));

    mockExecutor.tick();

    assertThat(podStates.values(),
        containsExactlyOnce(
            hasAssignment("test-resource")));
    assertThat(podStates.values(),
        containsExactlyOnce(
            hasAssignment("another-resource")));
  }

  @Test
  void mustNotReassignIfNothingHasChanged() throws IOException {
    setupMockPodCoordinationIO();

    MockExecutor mockExecutor = new MockExecutor();
    AgentCoordinator coordinator = new AgentCoordinator(
        podCoordinationIO, mock(Random.class), mockExecutor.getExecutor());

    Pod pod = createPod();

    setPodRequests(pod, "test-resource");

    coordinator.onAgentPodAdded(new AgentPodAdded(pod));

    mockExecutor.tick();

    assertThat(podStates, hasEntry(equalTo(pod), hasAssignment("test-resource")));

    mockExecutor.tick();
    verify(podCoordinationIO, times(1)).assign(any(), any());
  }

  @Test
  void mustUpdateLeadersWhenAgentIsDeleted() throws IOException {
    setupMockPodCoordinationIO();

    MockExecutor mockExecutor = new MockExecutor();
    AgentCoordinator coordinator = new AgentCoordinator(podCoordinationIO, mock(Random.class), mockExecutor.getExecutor());

    Pod[] pods = new Pod[] { createPod(), createPod() };

    setPodRequests(pods[0], "test-resource");
    setPodRequests(pods[1], "test-resource");

    coordinator.onAgentPodAdded(new AgentPodAdded(pods[0]));
    coordinator.onAgentPodAdded(new AgentPodAdded(pods[1]));

    mockExecutor.tick();

    //noinspection unchecked
    assertThat(podStates.values(), containsInAnyOrder(
        hasAssignment("test-resource"),
        not(hasAssignment("test-resource"))));

    Pod leader = getCurrentlyAssignedLeader("test-resource");
    coordinator.onAgentPodDeleted(new AgentPodDeleted(leader.getMetadata().getUid()));
    podStates.remove(leader);

    mockExecutor.tick();

    assertThat(podStates.values(),
        containsExactlyOnce(
            hasAssignment("test-resource")));
  }

  @Test
  void mustUpdateAssignmentsIfPollingFailsAtFirst() throws IOException {
    setupMockPodCoordinationIO();

    MockExecutor mockExecutor = new MockExecutor();
    AgentCoordinator coordinator = new AgentCoordinator(podCoordinationIO, mock(Random.class), mockExecutor.getExecutor());

    Pod[] pods = new Pod[] { createPod(), createPod() };

    setPodRequests(pods[0], "test-resource");
    setPodRequests(pods[1], "test-resource");

    doThrow(new IOException("failure"))
        .doAnswer(i -> getPodState(i.getArgument(0)))
        .when(podCoordinationIO).pollPod(any());

    coordinator.onAgentPodAdded(new AgentPodAdded(pods[0]));
    coordinator.onAgentPodAdded(new AgentPodAdded(pods[1]));

    mockExecutor.tick();
    mockExecutor.tick();

    assertThat(podStates.values(),
        containsExactlyOnce(
            hasAssignment("test-resource")));
  }

  @Test
  void mustUpdateAssignmentsForOtherPodsIfOnePodIsNotResponding() throws IOException {
    setupMockPodCoordinationIO();

    MockExecutor mockExecutor = new MockExecutor();
    AgentCoordinator coordinator = new AgentCoordinator(podCoordinationIO, mock(Random.class), mockExecutor.getExecutor());

    Pod[] pods = new Pod[] { createPod(), createPod() };

    setPodRequests(pods[0], "test-resource");
    doThrow(new IOException("failure"))
        .when(podCoordinationIO).pollPod(pods[1]);

    coordinator.onAgentPodAdded(new AgentPodAdded(pods[0]));
    coordinator.onAgentPodAdded(new AgentPodAdded(pods[1]));

    mockExecutor.tick();

    assertThat(podStates.get(pods[0]), hasAssignment("test-resource"));
    assertThat(podStates.get(pods[1]), not(hasAssignment("test-resource")));
  }

  @Test
  void mustUpdateAssignmentsForOtherPodsIfAssignmentFailsForOnePod() throws IOException {
    setupMockPodCoordinationIO();

    MockExecutor mockExecutor = new MockExecutor();
    AgentCoordinator coordinator = new AgentCoordinator(podCoordinationIO, mock(Random.class), mockExecutor.getExecutor());

    Pod[] pods = new Pod[] { createPod(), createPod() };

    setPodRequests(pods[0], "test-resource");
    setPodRequests(pods[1], "test-resource");

    coordinator.onAgentPodAdded(new AgentPodAdded(pods[0]));
    coordinator.onAgentPodAdded(new AgentPodAdded(pods[1]));

    mockExecutor.tick();

    Pod leader = getCurrentlyAssignedLeader("test-resource");

    doThrow(new IOException("failure"))
        .when(podCoordinationIO).pollPod(eq(leader));
    doThrow(new IOException("failure"))
        .when(podCoordinationIO).assign(eq(leader), any());

    mockExecutor.tick();

    Pod otherPod = pods[0].equals(leader) ? pods[1] : pods[0];
    assertThat(podStates.get(otherPod), hasAssignment("test-resource"));
  }

  @Test
  void mustRecoverIfAssignmentIsUpdatedOutOfBand() throws IOException {
    setupMockPodCoordinationIO();
    MockExecutor mockExecutor = new MockExecutor();

    AgentCoordinator coordinator = new AgentCoordinator(podCoordinationIO, mock(Random.class), mockExecutor.getExecutor());

    Pod[] pods = new Pod[] { createPod(), createPod() };

    setPodRequests(pods[0], "another-resource");
    setPodRequests(pods[1], "another-resource");

    coordinator.onAgentPodAdded(new AgentPodAdded(pods[0]));
    coordinator.onAgentPodAdded(new AgentPodAdded(pods[1]));

    mockExecutor.tick();

    Pod leader = getCurrentlyAssignedLeader("another-resource");

    // pod reports it does not have any assignments for some reason
    setPodAssignment(leader);

    mockExecutor.tick();

    assertThat(podStates.values(),
        containsExactlyOnce(
            hasAssignment("another-resource")));
  }

  @Test
  void shouldNotTryToAssignMoreThanOncePerTick() throws IOException {
    setupMockPodCoordinationIO();
    MockExecutor mockExecutor = new MockExecutor();

    AgentCoordinator coordinator = new AgentCoordinator(podCoordinationIO, mock(Random.class), mockExecutor.getExecutor());

    Pod pod = createPod();

    setPodRequests(pod, "test-resource");

    doThrow(new IOException("failure"))
        .when(podCoordinationIO).assign(any(), any());

    coordinator.onAgentPodAdded(new AgentPodAdded(pod));

    mockExecutor.tick();

    verify(podCoordinationIO, times(1)).assign(any(), any());
  }

  @Test
  void mustNotPollNonExistentPods() throws IOException {
    setupMockPodCoordinationIO();
    MockExecutor mockExecutor = new MockExecutor();

    AgentCoordinator coordinator = new AgentCoordinator(podCoordinationIO, mock(Random.class), mockExecutor.getExecutor());

    Pod pod1 = createPod();
    Pod pod2 = createPod();

    setPodRequests(pod1, "test-resource");
    setPodRequests(pod2, "test-resource");

    coordinator.onAgentPodAdded(new AgentPodAdded(pod1));
    coordinator.onAgentPodAdded(new AgentPodAdded(pod2));

    mockExecutor.tick();

    Mockito.clearInvocations(podCoordinationIO);

    coordinator.onAgentPodDeleted(new AgentPodDeleted(pod1.getMetadata().getUid()));

    mockExecutor.tick();

    verify(podCoordinationIO, times(1)).pollPod(pod2);
    verify(podCoordinationIO, atLeast(0)).assign(pod2, singleton("test-resource"));
    verifyNoMoreInteractions(podCoordinationIO);
  }

  @Test
  void mustHandleWhenPodIsDeletedDuringReconciliation() throws IOException {
    MockExecutor mockExecutor = new MockExecutor();

    AtomicReference<AgentCoordinator> coordinatorRef = new AtomicReference<>();

    Pod pod1 = createPod();
    Pod pod2 = createPod();

    setupMockPodCoordinationIO(() -> { },
        () -> {
          coordinatorRef.get().onAgentPodDeleted(new AgentPodDeleted(pod2.getMetadata().getUid()));
          podStates.remove(pod2);
        });

    AgentCoordinator coordinator = new AgentCoordinator(podCoordinationIO, mock(Random.class), mockExecutor.getExecutor());
    coordinatorRef.set(coordinator);

    setPodRequests(pod1, "test-resource");
    setPodRequests(pod2, "test-resource");

    coordinator.onAgentPodAdded(new AgentPodAdded(pod1));
    coordinator.onAgentPodAdded(new AgentPodAdded(pod2));

    mockExecutor.tick();

    assertThat(podStates.get(pod1), hasAssignment("test-resource"));
  }

  void setupMockPodCoordinationIO() throws IOException {
    setupMockPodCoordinationIO(() -> {
    }, () -> {
    });
  }

  void setupMockPodCoordinationIO(Runnable onAssignHook, Runnable onPollHook) throws IOException {
    podCoordinationIO = mock(PodCoordinationIO.class);

    podStates = new HashMap<>();
    doAnswer(i -> {
      setPodAssignment(i.getArgument(0), i.<Set<String>>getArgument(1));
      onAssignHook.run();
      return null;
    }).when(podCoordinationIO).assign(any(), any());

    when(podCoordinationIO.pollPod(any()))
        .thenAnswer(i -> {
          CoordinationRecord record = getPodState(i.getArgument(0));
          onPollHook.run();
          return record;
        });
  }

  CoordinationRecord getPodState(Pod pod) throws IOException {
    if (!podStates.containsKey(pod)) {
      throw new IOException("Pod " + pod.getMetadata().getName() + " does not exist");
    }
    return podStates.get(pod);
  }

  void setPodRequests(Pod pod, String... requested) {
    setPodRequests(pod, new HashSet<>(Arrays.asList(requested)));
  }

  void setPodRequests(Pod pod, Set<String> requested) {
    podStates.compute(pod, (k, v) -> {
      if (v == null) {
        return new CoordinationRecord(requested, Collections.emptySet());
      } else {
        return new CoordinationRecord(requested, v.getAssigned());
      }
    });
  }

  void setPodAssignment(Pod pod, String... assigned) throws IOException {
    setPodAssignment(pod, new HashSet<>(Arrays.asList(assigned)));
  }

  void setPodAssignment(Pod pod, Set<String> assigned) throws IOException {
    if (!podStates.containsKey(pod)) {
      throw new IOException("Pod " + pod.getMetadata().getName() + " does not exist");
    }
    podStates.compute(pod, (k, v) -> {
      if (v == null) {
        return new CoordinationRecord(Collections.emptySet(), assigned);
      } else {
        return new CoordinationRecord(v.getRequested(), assigned);
      }
    });
  }

  Pod getCurrentlyAssignedLeader(String resource) {
    return podStates.entrySet().stream()
        .filter(e -> e.getValue().getAssigned().contains(resource))
        .map(Map.Entry::getKey)
        .findFirst()
        .get();
  }

  private Pod createPod() {
    String uid = UUID.randomUUID().toString();
    return new PodBuilder()
        .withNewMetadata()
        .withUid(uid)
        .withName(uid.substring(0, 5))
        .endMetadata()
        .build();
  }

  static class MockExecutor {
    private ScheduledExecutorService executor;

    private List<Runnable> runnables = new ArrayList<>();

    MockExecutor() {
      executor = mock(ScheduledExecutorService.class);
      when(executor.scheduleWithFixedDelay(any(), anyLong(), anyLong(), any()))
          .thenAnswer(i -> {
            Runnable runnable = i.getArgument(0);
            runnables.add(runnable);
            return new TestFuture(runnable, () -> runnables.remove(runnable));
          });
    }

    ScheduledExecutorService getExecutor() {
      return executor;
    }
    void tick() {
      runnables.forEach(Runnable::run);
    }

  }

  static class TestFuture<V> extends FutureTask<V> implements ScheduledFuture<V> {

    private final Runnable onCancel;

    TestFuture(Runnable runnable, Runnable onCancel) {
      super(runnable, null);
      this.onCancel = onCancel;
    }

    @Override
    public boolean cancel(boolean mayInterruptIfRunning) {
      onCancel.run();
      return super.cancel(mayInterruptIfRunning);
    }

    @SuppressWarnings("NullableProblems")
    @Override
    public long getDelay(TimeUnit unit) {
      return 0;
    }

    @SuppressWarnings("NullableProblems")
    @Override
    public int compareTo(Delayed o) {
      return 0;
    }
  }

  static TypeSafeMatcher<CoordinationRecord> hasAssignment(String resource) {
    return new TypeSafeMatcher<CoordinationRecord>() {
      @Override
      protected boolean matchesSafely(CoordinationRecord item) {
        return item.getAssigned().contains(resource);
      }

      @Override
      public void describeTo(Description description) {
        description.appendText("has assignment " + resource);
      }
    };
  }

  static <T> TypeSafeMatcher<Iterable<T>> containsExactlyOnce(Matcher<T> matcher) {
    return new TypeSafeMatcher<Iterable<T>>() {
      @Override
      protected boolean matchesSafely(Iterable<T> items) {
        int count = 0;
        for (T item : items) {
          if (matcher.matches(item)) {
            count++;
          }
        }
        return count == 1;
      }

      @Override
      public void describeTo(Description description) {
        description.appendText("contains exactly one " + matcher);
      }
    };
  }
}
