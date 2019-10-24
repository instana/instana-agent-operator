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
  private final CommittedAssignments committedAssignments = CommittedAssignments.empty();
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

    schedule = executor.scheduleWithFixedDelay(this::pollAgentsAndAssignLeaders, 5, 5, TimeUnit.SECONDS);
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
    }
  }

  private void pollAgentsAndAssignLeaders() {
    try {
      Map<String, CoordinationRecord> polledState = pollPodsForResourceInterests();

      commitPolledState(polledState);

      Set<String> failedPods;
      do {
        DesiredAssignments desired = calculateAssignments(polledState);

        failedPods = assign(polledState.keySet(), desired);

        for (String failedPod : failedPods) {
          polledState.remove(failedPod);
        }
      } while (!failedPods.isEmpty());
    } catch (Exception e) {
      LOGGER.error("Exception polling agents and assigning leaders", e);
    }
  }

  private Map<String, CoordinationRecord> pollPodsForResourceInterests() {
    Map<String, Pod> pods = agentPods.get();
    Map<String, CoordinationRecord> resourcesByPod = new HashMap<>();

    for (Map.Entry<String, Pod> entry : pods.entrySet()) {
      try {
        CoordinationRecord record = podCoordinationIO.pollPod(entry.getValue());

        resourcesByPod.put(entry.getKey(), record);
      } catch (IOException e) {
        LOGGER.debug("Failed to poll for requested leaderships from {}", entry.getValue().getMetadata().getName());
      }
    }

    return resourcesByPod;
  }

  private void commitPolledState(Map<String, CoordinationRecord> polledState) {
    for (Map.Entry<String, CoordinationRecord> pod : polledState.entrySet()) {
      if (pod.getValue() == null || pod.getValue().getAssigned() == null) {
        committedAssignments.commit(pod.getKey(), Collections.emptySet());
      } else {
        committedAssignments.commit(pod.getKey(), pod.getValue().getAssigned());
      }
    }
  }

  private DesiredAssignments calculateAssignments(Map<String, CoordinationRecord> polledState) {
    Map<String, Set<String>> podsByResource = new HashMap<>();

    for (Map.Entry<String, CoordinationRecord> pod : polledState.entrySet()) {
      pod.getValue().getRequested().forEach(resource -> podsByResource.compute(resource, (r, pods) -> {
        if (pods == null) {
          pods = new HashSet<>();
        }
        pods.add(pod.getKey());
        return pods;
      }));
    }

    HashMap<String, String> desired = new HashMap<>();
    for (Map.Entry<String, Set<String>> resource : podsByResource.entrySet()) {
      String currentLeaderPod = committedAssignments.leaderForResource(resource.getKey());
      Set<String> possiblePods = podsByResource.get(resource.getKey());
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

  private Set<String> assign(Set<String> polledPods, DesiredAssignments desired) {
    Map<String, Pod> pods = new HashMap<>(agentPods.get());
    pods.keySet().retainAll(polledPods);
    Set<String> failedPods = new HashSet<>();

    for (Map.Entry<String, Pod> pod : pods.entrySet()) {
      Set<String> previous = committedAssignments.resourcesAssignedToPod(pod.getKey());
      Set<String> updated = desired.resourcesAssignedToPod(pod.getKey());
      if (updated.equals(previous)) {
        continue;
      }
      try {
        podCoordinationIO.assign(pod.getValue(), updated);
        committedAssignments.commit(pod.getKey(), updated);
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

  static class CommittedAssignments {
    final Map<String, Set<String>> status;

    CommittedAssignments(Map<String, Set<String>> status) {
      this.status = status;
    }

    static CommittedAssignments empty() {
      return new CommittedAssignments(new HashMap<>());
    }

    String leaderForResource(String resource) {
      for (Map.Entry<String, Set<String>> pod : status.entrySet()) {
        if (pod.getValue().contains(resource)) {
          return pod.getKey();
        }
      }
      return null;
    }

    Set<String> resourcesAssignedToPod(String podUid) {
      return status.getOrDefault(podUid, Collections.emptySet());
    }

    void commit(String pod, Set<String> resources) {
      status.put(pod, resources);
    }
  }

  static class DesiredAssignments {
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
