package com.instana.operator.coordination;

import com.instana.operator.events.AgentPodAdded;
import com.instana.operator.events.AgentPodDeleted;
import io.fabric8.kubernetes.api.model.Pod;
import io.fabric8.kubernetes.api.model.PodBuilder;
import org.hamcrest.Description;
import org.hamcrest.Matcher;
import org.hamcrest.TypeSafeMatcher;
import org.junit.jupiter.api.Test;

import java.io.IOException;
import java.util.ArrayList;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.UUID;
import java.util.concurrent.ScheduledExecutorService;

import static java.util.Collections.emptySet;
import static java.util.Collections.singleton;
import static org.hamcrest.MatcherAssert.assertThat;
import static org.hamcrest.Matchers.containsInAnyOrder;
import static org.hamcrest.Matchers.equalTo;
import static org.hamcrest.Matchers.hasEntry;
import static org.hamcrest.Matchers.not;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyLong;
import static org.mockito.Mockito.doAnswer;
import static org.mockito.Mockito.doThrow;
import static org.mockito.Mockito.eq;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.times;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class AgentCoordinatorTest {

  Map<Pod, CoordinationRecord> podStates;
  PodCoordinationIO podCoordinationIO;

  @Test
  void mustAssignLeaderForPodBasedOnRequestedResources() throws IOException {
    setupMockPodCoordinationIO();

    MockExecutor mockExecutor = new MockExecutor();
    AgentCoordinator coordinator = new AgentCoordinator(
        podCoordinationIO, mockExecutor.getExecutor());

    Pod[] pods = new Pod[]{createPod(), createPod(), createPod()};

    setPodRequests(pods[0], singleton("test-resource"));
    setPodRequests(pods[1], singleton("test-resource"));
    setPodRequests(pods[2], singleton("test-resource"));

    coordinator.onAgentPodAdded(new AgentPodAdded(pods[0]));
    coordinator.onAgentPodAdded(new AgentPodAdded(pods[1]));
    coordinator.onAgentPodAdded(new AgentPodAdded(pods[2]));

    mockExecutor.tick();

    assertThat(podStates.values(),
        containsExactlyOnce(
            hasAssignment("test-resource")));
  }

  @Test
  void mustNotReassignIfNothingHasChanged() throws IOException {
    setupMockPodCoordinationIO();

    MockExecutor mockExecutor = new MockExecutor();
    AgentCoordinator coordinator = new AgentCoordinator(
        podCoordinationIO, mockExecutor.getExecutor());

    Pod pod = createPod();

    setPodRequests(pod, singleton("test-resource"));

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
    AgentCoordinator coordinator = new AgentCoordinator(podCoordinationIO, mockExecutor.getExecutor());

    Pod[] pods = new Pod[] { createPod(), createPod() };

    setPodRequests(pods[0], singleton("test-resource"));
    setPodRequests(pods[1], singleton("test-resource"));

    coordinator.onAgentPodAdded(new AgentPodAdded(pods[0]));
    coordinator.onAgentPodAdded(new AgentPodAdded(pods[1]));

    mockExecutor.tick();

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
    AgentCoordinator coordinator = new AgentCoordinator(podCoordinationIO, mockExecutor.getExecutor());

    Pod[] pods = new Pod[] { createPod(), createPod() };

    setPodRequests(pods[0], singleton("test-resource"));
    setPodRequests(pods[1], singleton("test-resource"));

    when(podCoordinationIO.pollPod(any()))
        .thenThrow(new IOException("failure"))
        .thenAnswer(i -> podStates.get(i.<Pod>getArgument(0)));

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
    AgentCoordinator coordinator = new AgentCoordinator(podCoordinationIO, mockExecutor.getExecutor());

    Pod[] pods = new Pod[] { createPod(), createPod() };

    setPodRequests(pods[0], singleton("test-resource"));
    when(podCoordinationIO.pollPod(pods[1]))
        .thenThrow(new IOException("failure"));

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
    AgentCoordinator coordinator = new AgentCoordinator(podCoordinationIO, mockExecutor.getExecutor());

    Pod[] pods = new Pod[] { createPod(), createPod() };

    setPodRequests(pods[0], singleton("test-resource"));
    setPodRequests(pods[1], singleton("test-resource"));

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

    AgentCoordinator coordinator = new AgentCoordinator(podCoordinationIO, mockExecutor.getExecutor());

    Pod[] pods = new Pod[] { createPod(), createPod() };

    setPodRequests(pods[0], singleton("test-resource"));
    setPodRequests(pods[1], singleton("test-resource"));

    coordinator.onAgentPodAdded(new AgentPodAdded(pods[0]));
    coordinator.onAgentPodAdded(new AgentPodAdded(pods[1]));

    mockExecutor.tick();

    Pod leader = getCurrentlyAssignedLeader("test-resource");

    // pod reports it does not have any assignments for some reason
    setPodAssignment(leader, emptySet());

    mockExecutor.tick();

    assertThat(podStates.values(),
        containsExactlyOnce(
            hasAssignment("test-resource")));
  }

  @Test
  void shouldNotTryToAssignMoreThanOncePerTick() throws IOException {
    setupMockPodCoordinationIO();
    MockExecutor mockExecutor = new MockExecutor();

    AgentCoordinator coordinator = new AgentCoordinator(podCoordinationIO, mockExecutor.getExecutor());

    Pod pod = createPod();

    setPodRequests(pod, singleton("test-resource"));

    doThrow(new IOException("failure"))
        .when(podCoordinationIO).assign(any(), any());

    coordinator.onAgentPodAdded(new AgentPodAdded(pod));

    mockExecutor.tick();

    verify(podCoordinationIO, times(1)).assign(any(), any());
  }

  void setupMockPodCoordinationIO() throws IOException {
    podCoordinationIO = mock(PodCoordinationIO.class);

    podStates = new HashMap<>();
    doAnswer(i -> setPodAssignment(i.getArgument(0), i.getArgument(1)))
        .when(podCoordinationIO).assign(any(), any());

    when(podCoordinationIO.pollPod(any()))
        .thenAnswer(i -> podStates.get(i.<Pod>getArgument(0)));
  }

  CoordinationRecord setPodRequests(Pod pod, Set<String> requested) {
    return podStates.compute(pod, (k, v) -> {
      if (v == null) {
        return new CoordinationRecord(requested, Collections.emptySet());
      } else {
        return new CoordinationRecord(requested, v.getAssigned());
      }
    });
  }

  CoordinationRecord setPodAssignment(Pod pod, Set<String> assigned) {
    return podStates.compute(pod, (k, v) -> {
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

    public MockExecutor() {
      executor = mock(ScheduledExecutorService.class);
      when(executor.scheduleWithFixedDelay(any(), anyLong(), anyLong(), any()))
          .thenAnswer(i -> {
            Runnable runnable = i.getArgument(0);
            runnables.add(runnable);
            return null;
          });
    }

    public ScheduledExecutorService getExecutor() {
      return executor;
    }
    void tick() {
      runnables.forEach(Runnable::run);
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
