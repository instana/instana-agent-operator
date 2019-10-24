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
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.ScheduledFuture;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicReference;

import static com.instana.operator.ExecutorProducer.AGENT_COORDINATOR_POLL;

@ApplicationScoped
class AgentCoordinator {
  private static final Logger LOGGER = LoggerFactory.getLogger(AgentCoordinator.class);
  private final AtomicReference<Map<String, Pod>> agentPods = new AtomicReference<>(Collections.emptyMap());
  private final PodCoordinationIO podCoordinationIO;
  private final Random random = new Random();

  private ScheduledFuture<?> schedule;

  @Inject
  AgentCoordinator(PodCoordinationIO podCoordinationIO,
                   @Named(AGENT_COORDINATOR_POLL) ScheduledExecutorService executor) {
    this.podCoordinationIO = podCoordinationIO;
    this.executor = executor;
  }

  private ScheduledExecutorService executor;

  void onAgentPodAdded(@ObservesAsync AgentPodAdded event) {
    LOGGER.info("Pod {} added", event.getPod().getMetadata().getName());
    Map<String, Pod> previous = agentPods.getAndUpdate(current -> {
      HashMap<String, Pod> update = new HashMap<>(current);
      update.put(event.getPod().getMetadata().getUid(), event.getPod());
      return update;
    });

    if (!previous.isEmpty()) {
      return;
    }

    if (schedule == null) {
      schedule = executor.scheduleWithFixedDelay(this::pollAgentsAndAssignLeaders, 5, 5, TimeUnit.SECONDS);
    }
  }

  void onAgentPodDeleted(@ObservesAsync AgentPodDeleted event) {
    LOGGER.info("Pod {} deleted", getReadablePodName(event.getUid()));
    Map<String, Pod> pods = agentPods.updateAndGet(current -> {
      HashMap<String, Pod> update = new HashMap<>(current);
      update.remove(event.getUid());
      return update;
    });

    if (pods.isEmpty()) {
      schedule.cancel(false);
      schedule = null;
    }
  }

  private void pollAgentsAndAssignLeaders() {
    try {
      Map<String, Pod> activePods = agentPods.get();
      Set<String> failedPods;
      do {
        LeadershipStatus leadershipStatus = pollForLeadershipStatus(activePods);

        DesiredAssignments desired = calculateAssignments(leadershipStatus);

        failedPods = assign(leadershipStatus, desired);

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
      String currentLeaderPod = leadershipStatus.leaderForResource(resource.getKey());
      Set<String> possiblePods = resourceRequests.get(resource.getKey());
      if (currentLeaderPod != null && possiblePods.contains(currentLeaderPod)) {
        desired.put(resource.getKey(), currentLeaderPod);
      } else {
        int selectedPodIndex = random.nextInt(possiblePods.size());
        List<String> podList = new ArrayList<>(possiblePods);
        desired.put(resource.getKey(), podList.get(selectedPodIndex));
      }
    }

    return new DesiredAssignments(desired);
  }

  private Set<String> assign(LeadershipStatus leadershipStatus, DesiredAssignments desired) {
    Map<String, Pod> pods = new HashMap<>(agentPods.get());
    pods.keySet().retainAll(leadershipStatus.responsivePods());
    Set<String> failedPods = new HashSet<>();

    for (Map.Entry<String, Pod> pod : pods.entrySet()) {
      Set<String> previous = leadershipStatus.resourcesAssignedToPod(pod.getKey());
      Set<String> updated = desired.resourcesAssignedToPod(pod.getKey());
      if (updated.equals(previous)) {
        continue;
      }
      try {
        podCoordinationIO.assign(pod.getValue(), updated);
        LOGGER.info("Assigned leadership of {} to {}", updated, pod.getValue().getMetadata().getName());
      } catch (IOException e) {
        LOGGER.warn("Failed to assign leading resources to {}: {}", pod.getValue().getMetadata().getName(), e.getMessage());
        failedPods.add(pod.getKey());
      }
    }
    return failedPods;
  }

  private String getReadablePodName(String uid) {
    return Optional.ofNullable(
        agentPods.get().get(uid))
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

    String leaderForResource(String resource) {
      for (Map.Entry<String, CoordinationRecord> pod : status.entrySet()) {
        if (pod.getValue() == null || pod.getValue().getAssigned() == null) {
          continue;
        }
        if (pod.getValue().getAssigned().contains(resource)) {
          return pod.getKey();
        }
      }
      return null;
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
      Set<String> assignedResources = new HashSet<>();
      for (Map.Entry<String, String> resource : spec.entrySet()) {
        if (resource.getValue().equals(podUid)) {
          assignedResources.add(resource.getKey());
        }
      }
      return assignedResources;
    }
  }
}
