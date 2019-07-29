package com.instana.operator.coordination;

import com.instana.operator.events.AgentPodAdded;
import com.instana.operator.events.AgentPodDeleted;
import io.fabric8.kubernetes.api.model.Pod;
import io.fabric8.kubernetes.api.model.PodBuilder;
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

import static org.hamcrest.MatcherAssert.assertThat;
import static org.hamcrest.Matchers.contains;
import static org.hamcrest.Matchers.empty;
import static org.hamcrest.Matchers.equalTo;
import static org.hamcrest.Matchers.hasEntry;
import static org.hamcrest.Matchers.hasKey;
import static org.hamcrest.Matchers.hasSize;
import static org.hamcrest.Matchers.hasValue;
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

  Map<Pod, Set<String>> assignments;
  PodCoordinationIO podCoordinationIO;

  @Test
  void mustAssignLeaderForPodBasedOnRequestedResources() throws IOException {
    setupMockPodCoordinationIO();

    MockExecutor mockExecutor = new MockExecutor();
    AgentCoordinator coordinator = new AgentCoordinator(
        podCoordinationIO, mockExecutor.getExecutor());

    Pod[] pods = new Pod[]{createPod(), createPod(), createPod()};

    coordinator.onAgentPodAdded(new AgentPodAdded(pods[0]));
    coordinator.onAgentPodAdded(new AgentPodAdded(pods[1]));
    coordinator.onAgentPodAdded(new AgentPodAdded(pods[2]));

    mockExecutor.tick();

    List<String> assignedResources = new ArrayList<>();
    assignments.values().forEach(assignedResources::addAll);

    assertThat(assignedResources, hasSize(1));
    assertThat(assignedResources, contains("test-resource"));
  }

  @Test
  void mustNotReassignIfNothingHasChanged() throws IOException {
    setupMockPodCoordinationIO();

    MockExecutor mockExecutor = new MockExecutor();
    AgentCoordinator coordinator = new AgentCoordinator(
        podCoordinationIO, mockExecutor.getExecutor());

    Pod pod = createPod();

    coordinator.onAgentPodAdded(new AgentPodAdded(pod));

    mockExecutor.tick();

    assertThat(assignments, hasEntry(equalTo(pod), contains("test-resource")));
    assignments.clear();
    when(podCoordinationIO.pollPod(pod))
        .thenReturn(new CoordinationRecord(Collections.singleton("test-resource"), Collections.singleton("test-resource")));

    mockExecutor.tick();
    assertThat(assignments.entrySet(), empty());
  }

  @Test
  void mustUpdateLeadersWhenAgentIsDeleted() throws IOException {
    setupMockPodCoordinationIO();

    MockExecutor mockExecutor = new MockExecutor();
    AgentCoordinator coordinator = new AgentCoordinator(podCoordinationIO, mockExecutor.getExecutor());

    Pod[] pods = new Pod[] { createPod(), createPod() };

    coordinator.onAgentPodAdded(new AgentPodAdded(pods[0]));
    coordinator.onAgentPodAdded(new AgentPodAdded(pods[1]));

    mockExecutor.tick();

    assertThat(assignments, hasValue(contains("test-resource")));

    Pod leader = assignments.entrySet().stream()
        .filter(e -> e.getValue().contains("test-resource"))
        .map(Map.Entry::getKey)
        .findFirst()
        .get();

    assignments.clear();

    coordinator.onAgentPodDeleted(new AgentPodDeleted(leader.getMetadata().getUid()));

    mockExecutor.tick();

    assertThat(assignments, hasValue(contains("test-resource")));
    assertThat(assignments, not(hasKey(leader.getMetadata().getUid())));
  }

  @Test
  void mustUpdateAssignmentsIfPollingFailsAtFirst() throws IOException {
    setupMockPodCoordinationIO();

    MockExecutor mockExecutor = new MockExecutor();
    AgentCoordinator coordinator = new AgentCoordinator(podCoordinationIO, mockExecutor.getExecutor());

    Pod[] pods = new Pod[] { createPod(), createPod() };

    when(podCoordinationIO.pollPod(any()))
        .thenThrow(new IOException("failure"))
        .thenAnswer(i -> {
          Pod pod = i.getArgument(0);
          return new CoordinationRecord(Collections.singleton("test-resource"), assignments.get(pod));
        });

    coordinator.onAgentPodAdded(new AgentPodAdded(pods[0]));
    coordinator.onAgentPodAdded(new AgentPodAdded(pods[1]));

    mockExecutor.tick();
    mockExecutor.tick();

    List<String> assignedResources = new ArrayList<>();
    assignments.values().forEach(assignedResources::addAll);

    assertThat(assignedResources, hasSize(1));
    assertThat(assignedResources, contains("test-resource"));
  }

  @Test
  void mustUpdateAssignmentsForOtherPodsIfOnePodIsNotResponding() throws IOException {
    setupMockPodCoordinationIO();

    MockExecutor mockExecutor = new MockExecutor();
    AgentCoordinator coordinator = new AgentCoordinator(podCoordinationIO, mockExecutor.getExecutor());

    Pod[] pods = new Pod[] { createPod(), createPod() };

    when(podCoordinationIO.pollPod(pods[0]))
        .thenReturn(new CoordinationRecord(Collections.singleton("test-resource"), Collections.emptySet()));

    when(podCoordinationIO.pollPod(pods[1]))
        .thenThrow(new IOException("failure"));

    coordinator.onAgentPodAdded(new AgentPodAdded(pods[0]));
    coordinator.onAgentPodAdded(new AgentPodAdded(pods[1]));

    mockExecutor.tick();

    List<String> assignedResources = new ArrayList<>();
    assignments.values().forEach(assignedResources::addAll);

    assertThat(assignedResources, hasSize(1));
    assertThat(assignedResources, contains("test-resource"));
  }

  @Test
  void mustUpdateAssignmentsForOtherPodsIfAssignmentFailsForOnePod() throws IOException {
    setupMockPodCoordinationIO();

    MockExecutor mockExecutor = new MockExecutor();
    AgentCoordinator coordinator = new AgentCoordinator(podCoordinationIO, mockExecutor.getExecutor());

    Pod[] pods = new Pod[] { createPod(), createPod() };

    when(podCoordinationIO.pollPod(any()))
        .thenReturn(new CoordinationRecord(Collections.singleton("test-resource"), Collections.emptySet()));

    HashMap<Pod, Set<String>> assignments = new HashMap<>();
    doAnswer(i -> assignments.put(i.getArgument(0), i.getArgument(1)))
        .when(podCoordinationIO).assign(eq(pods[0]), any());

    doThrow(new IOException("failure"))
        .when(podCoordinationIO).assign(eq(pods[1]), any());

    coordinator.onAgentPodAdded(new AgentPodAdded(pods[0]));
    coordinator.onAgentPodAdded(new AgentPodAdded(pods[1]));

    mockExecutor.tick();

    List<String> assignedResources = new ArrayList<>();
    assignments.values().forEach(assignedResources::addAll);

    assertThat(assignedResources, hasSize(1));
    assertThat(assignedResources, contains("test-resource"));
  }

  @Test
  void mustRecoverIfAssignmentIsUpdatedOutOfBand() throws IOException {
    setupMockPodCoordinationIO();
    MockExecutor mockExecutor = new MockExecutor();

    AgentCoordinator coordinator = new AgentCoordinator(podCoordinationIO, mockExecutor.getExecutor());

    Pod[] pods = new Pod[] { createPod(), createPod() };

    coordinator.onAgentPodAdded(new AgentPodAdded(pods[0]));
    coordinator.onAgentPodAdded(new AgentPodAdded(pods[1]));

    mockExecutor.tick();

    Pod leader = assignments.entrySet().stream()
        .filter(e -> e.getValue().contains("test-resource"))
        .map(Map.Entry::getKey)
        .findFirst()
        .get();

    assignments.clear();

    // pod reports it does not have any assignments for some reason
    when(podCoordinationIO.pollPod(leader))
        .thenReturn(new CoordinationRecord(Collections.singleton("test-resource"), Collections.emptySet()));

    mockExecutor.tick();

    List<String> assignedResources = new ArrayList<>();
    assignments.values().forEach(assignedResources::addAll);

    assertThat(assignedResources, hasSize(1));
    assertThat(assignedResources, contains("test-resource"));
  }

  @Test
  void shouldNotTryToAssignMoreThanOncePerTick() throws IOException {
    setupMockPodCoordinationIO();
    MockExecutor mockExecutor = new MockExecutor();

    AgentCoordinator coordinator = new AgentCoordinator(podCoordinationIO, mockExecutor.getExecutor());

    Pod pod = createPod();

    doThrow(new IOException("failure"))
        .when(podCoordinationIO).assign(any(), any());

    coordinator.onAgentPodAdded(new AgentPodAdded(pod));

    mockExecutor.tick();

    verify(podCoordinationIO, times(1)).assign(any(), any());
  }

  void setupMockPodCoordinationIO() throws IOException {
    podCoordinationIO = mock(PodCoordinationIO.class);

    when(podCoordinationIO.pollPod(any()))
        .thenAnswer(i -> {
          Pod pod = i.getArgument(0);
          return new CoordinationRecord(Collections.singleton("test-resource"), assignments.get(pod));
        });

    assignments = new HashMap<>();
    doAnswer(i -> assignments.put(i.getArgument(0), i.getArgument(1)))
        .when(podCoordinationIO).assign(any(), any());
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
}
