package com.instana.operator;

import com.instana.operator.cache.Cache;
import com.instana.operator.cache.CacheService;
import com.instana.operator.events.AgentPodAdded;
import com.instana.operator.events.AgentPodDeleted;
import com.instana.operator.events.DaemonSetAdded;
import com.instana.operator.events.DaemonSetDeleted;
import io.fabric8.kubernetes.api.model.Pod;
import io.fabric8.kubernetes.api.model.PodList;
import io.fabric8.kubernetes.api.model.apps.DaemonSet;
import io.fabric8.kubernetes.client.DefaultKubernetesClient;
import io.reactivex.disposables.Disposable;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.event.Event;
import javax.enterprise.event.NotificationOptions;
import javax.enterprise.event.ObservesAsync;
import javax.inject.Inject;
import java.util.HashSet;
import java.util.Optional;
import java.util.Set;

import static com.instana.operator.util.ResourceUtils.hasOwner;
import static com.instana.operator.util.ResourceUtils.isRunning;
import static com.instana.operator.util.ResourceUtils.name;

@ApplicationScoped
public class DaemonSetWatcher {

  @Inject
  DefaultKubernetesClient defaultClient;
  @Inject
  CacheService cacheService;
  @Inject
  Event<AgentPodAdded> agentPodAddedEvent;
  @Inject
  Event<AgentPodDeleted> agentPodDeletedEvent;
  @Inject
  NotificationOptions asyncSerial;
  @Inject
  FatalErrorHandler fatalErrorHandler;

  private final Logger LOGGER = LoggerFactory.getLogger(DaemonSetWatcher.class);
  private final Set<String> knownUIDs = new HashSet<>();
  private Disposable watch = null;
  private DaemonSet daemonSet;

  // Handles Kubernetes events (received in the k8s-handler thread),
  // and creates business events (scheduled in the cdi-handler thread).
  // That way the actual business logic (AgentLeaderManager) does not need to be thread safe.
  public void daemonSetAdded(@ObservesAsync DaemonSetAdded event) {
    if (daemonSet != null) {
      if (daemonSet.getMetadata().getUid().equals(event.getDaemonSet().getMetadata().getUid())) {
        return; // already watching
      } else {
        LOGGER.error("Received ADDED event for DaemonSet " + name(event.getDaemonSet()) + " but I'm already" +
            " watching " + name(daemonSet) + ".");
        fatalErrorHandler.systemExit(-1);
      }
    }
    daemonSetDeleted(null);
    daemonSet = event.getDaemonSet();
    String namespace = daemonSet.getMetadata().getNamespace();
    Cache<Pod, PodList> podCache = cacheService.newCache(Pod.class, PodList.class);
    LOGGER.info("Looking for agent Pods in DaemonSet " + name(daemonSet) + "...");
    watch = podCache.listThenWatch(defaultClient.inNamespace(namespace).pods()).subscribe(
        e -> {
          Optional<Pod> pod = podCache.get(e.getUid())
              .filter(hasOwner(daemonSet))
              .filter(isRunning());
          if (pod.isPresent()) {
            if (knownUIDs.add(e.getUid())) {
              agentPodAddedEvent.fireAsync(new AgentPodAdded(pod.get()), asyncSerial)
                  .exceptionally(fatalErrorHandler::logAndExit);
            }
          } else {
            if (knownUIDs.remove(e.getUid())) {
              agentPodDeletedEvent.fireAsync(new AgentPodDeleted(e.getUid()), asyncSerial)
                  .exceptionally(fatalErrorHandler::logAndExit);
            }
          }
        });
  }

  public void daemonSetDeleted(@ObservesAsync DaemonSetDeleted event) {
    daemonSet = null;
    if (watch != null) {
      watch.dispose();
      watch = null;
    }
    for (String uid : knownUIDs) {
      agentPodDeletedEvent.fireAsync(new AgentPodDeleted(uid), asyncSerial)
          .exceptionally(fatalErrorHandler::logAndExit);
    }
    knownUIDs.clear();
  }
}
