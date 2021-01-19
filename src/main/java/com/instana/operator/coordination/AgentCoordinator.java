/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc.
 */
package com.instana.operator.coordination;

import com.instana.operator.events.AgentPodAdded;
import com.instana.operator.events.AgentPodDeleted;
import io.fabric8.kubernetes.api.model.Pod;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.event.ObservesAsync;
import javax.inject.Inject;
import javax.inject.Named;
import java.io.IOException;
import java.util.ArrayList;
import java.util.Collections;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.Random;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.ScheduledFuture;
import java.util.concurrent.TimeUnit;
import java.util.stream.Collectors;

import static com.instana.operator.ExecutorProducer.AGENT_COORDINATOR_POLL;

@ApplicationScoped
class AgentCoordinator {
  private static final Logger LOGGER = LoggerFactory.getLogger(AgentCoordinator.class);
  private final Map<String, Pod> agentPods = new ConcurrentHashMap<>();
  private final PodCoordinationIO podCoordinationIO;
  private final Random random;

  private ScheduledFuture<?> schedule;

  @Inject
  AgentCoordinator(PodCoordinationIO podCoordinationIO,
                   Random random,
                   @Named(AGENT_COORDINATOR_POLL) ScheduledExecutorService executor) {
    this.podCoordinationIO = podCoordinationIO;
    this.random = random;
    this.executor = executor;
  }

  private ScheduledExecutorService executor;

  void onAgentPodAdded(@ObservesAsync AgentPodAdded event) {
    LOGGER.info("Pod {} added", event.getPod().getMetadata().getName());
    agentPods.put(event.getPod().getMetadata().getUid(), event.getPod());

    if (schedule == null) {
      schedule = executor.scheduleWithFixedDelay(this::pollAgentsAndAssignLeaders, 5, 5, TimeUnit.SECONDS);
    }
  }

  void onAgentPodDeleted(@ObservesAsync AgentPodDeleted event) {
    LOGGER.info("Pod {} deleted", getReadablePodName(event.getUid()));
    agentPods.remove(event.getUid());

    if (agentPods.isEmpty()) {
      schedule.cancel(false);
      schedule = null;
    }
  }

  private void pollAgentsAndAssignLeaders() {
    try {
      Map<String, Pod> activePods = new HashMap<>(agentPods);
      Set<String> failedPods;
      do {
        LeadershipStatus leadershipStatus = pollForLeadershipStatus(activePods);

        DesiredAssignments desired = calculateAssignments(leadershipStatus);

        failedPods = assign(activePods, leadershipStatus, desired);

        for (String failedPod : failedPods) {
          activePods.remove(failedPod);
        }
      } while (!failedPods.isEmpty());
    } catch (Exception e) {
      LOGGER.error("Exception polling agents and assigning leaders", e);
    }
  }

  private LeadershipStatus pollForLeadershipStatus(Map<String, Pod> pods) {
    Map<String, CoordinationRecord> resourcesByPod = new HashMap<>();

    for (Map.Entry<String, Pod> entry : pods.entrySet()) {
      try {
        CoordinationRecord record = podCoordinationIO.pollPod(entry.getValue());

        resourcesByPod.put(entry.getKey(), record);
      } catch (IOException e) {
        LOGGER.debug("Failed to poll for requested leaderships from {}", entry.getValue().getMetadata().getName());
      }
    }

    return new LeadershipStatus(resourcesByPod);
  }

  private DesiredAssignments calculateAssignments(LeadershipStatus leadershipStatus) {
    Map<String, Set<String>> resourceRequests = leadershipStatus.getResourceRequests();

    HashMap<String, String> desired = new HashMap<>();
    for (Map.Entry<String, Set<String>> resource : resourceRequests.entrySet()) {
      Optional<String> currentLeaderPod = leadershipStatus.leaderForResource(resource.getKey());
      Set<String> possiblePods = resource.getValue();
      desired.put(
          resource.getKey(),
          currentLeaderPod
              .orElseGet(() -> selectRandomPod(possiblePods)));
    }

    return new DesiredAssignments(desired);
  }

  private Set<String> assign(Map<String, Pod> pods, LeadershipStatus leadershipStatus, DesiredAssignments desired) {
    Set<String> podIds = leadershipStatus.responsivePods();
    Set<String> failedPods = new HashSet<>();

    for (String podId : podIds) {
      Pod pod = pods.get(podId);
      Set<String> previous = leadershipStatus.resourcesAssignedToPod(podId);
      Set<String> updated = desired.resourcesAssignedToPod(podId);
      if (updated.equals(previous)) {
        continue;
      }
      try {
        podCoordinationIO.assign(pod, updated);
        LOGGER.info("Assigned leadership of {} to {}", updated, pod.getMetadata().getName());
      } catch (IOException e) {
        LOGGER.warn("Failed to assign leading resources to {}: {}", getReadablePodName(podId), e.getMessage());
        failedPods.add(podId);
      }
    }
    return failedPods;
  }

  private String selectRandomPod(Set<String> possiblePods) {
    int selectedPodIndex = random.nextInt(possiblePods.size());
    List<String> podList = new ArrayList<>(possiblePods);
    return podList.get(selectedPodIndex);
  }

  private String getReadablePodName(String uid) {
    return Optional.ofNullable(
        agentPods.get(uid))
        .map(p -> p.getMetadata().getName())
        .orElse(uid);
  }

  private static class LeadershipStatus {
    final Map<String, CoordinationRecord> status;

    LeadershipStatus(Map<String, CoordinationRecord> status) {
      this.status = status;
    }

    Map<String, Set<String>> getResourceRequests() {
      Map<String, Set<String>> resourceRequests = new HashMap<>();

      for (Map.Entry<String, CoordinationRecord> pod : status.entrySet()) {
        for (String resource : pod.getValue().getRequested()) {
          resourceRequests.compute(resource, (k, prev) -> {
            Set<String> pods = Optional.ofNullable(prev)
                .orElseGet(HashSet::new);
            pods.add(pod.getKey());
            return pods;
          });
        }
      }

      return resourceRequests;
    }

    Set<String> resourcesAssignedToPod(String podUid) {
      return Optional.ofNullable(status.get(podUid))
          .map(CoordinationRecord::getAssigned)
          .orElse(Collections.emptySet());
    }

    Optional<String> leaderForResource(String resource) {
      return status.entrySet()
          .stream()
          .filter(e -> e.getValue() != null && e.getValue().getAssigned() != null)
          .filter(e -> e.getValue().getAssigned().contains(resource))
          .map(Map.Entry::getKey)
          .findFirst();
    }

    Set<String> responsivePods() {
      return status.keySet();
    }

  }

  private static class DesiredAssignments {
    final Map<String, String> spec;

    DesiredAssignments(Map<String, String> spec) {
      this.spec = spec;
    }

    Set<String> resourcesAssignedToPod(String podUid) {
      return this.spec.entrySet()
          .stream()
          .filter(e -> e.getValue().equals(podUid))
          .map(Map.Entry::getKey)
          .collect(Collectors.toSet());
    }
  }
}
