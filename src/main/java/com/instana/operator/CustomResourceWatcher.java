package com.instana.operator;

import com.instana.operator.cache.Cache;
import com.instana.operator.cache.CacheService;
import com.instana.operator.client.KubernetesClientProducer;
import com.instana.operator.customresource.DoneableInstanaAgent;
import com.instana.operator.customresource.InstanaAgent;
import com.instana.operator.customresource.InstanaAgentList;
import com.instana.operator.env.NamespaceProducer;
import com.instana.operator.events.CustomResourceAdded;
import com.instana.operator.events.CustomResourceDeleted;
import com.instana.operator.events.CustomResourceModified;
import com.instana.operator.events.CustomResourceOtherInstanceAdded;
import com.instana.operator.events.OperatorLeaderElected;
import io.fabric8.kubernetes.client.Watch;
import io.fabric8.kubernetes.client.Watcher;
import io.fabric8.kubernetes.client.dsl.FilterWatchListMultiDeletable;
import io.fabric8.kubernetes.client.dsl.MixedOperation;
import io.fabric8.kubernetes.client.dsl.Resource;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.event.Event;
import javax.enterprise.event.NotificationOptions;
import javax.enterprise.event.ObservesAsync;
import javax.inject.Inject;
import javax.inject.Named;
import java.util.ArrayList;
import java.util.List;
import java.util.Optional;
import java.util.Set;
import java.util.concurrent.atomic.AtomicReference;

@ApplicationScoped
public class CustomResourceWatcher {

  @Inject
  CacheService cacheService;
  @Inject
  MixedOperation<InstanaAgent, InstanaAgentList, DoneableInstanaAgent, Resource<InstanaAgent, DoneableInstanaAgent>> client;
  @Inject
  @Named(NamespaceProducer.TARGET_NAMESPACES)
  Set<String> targetNamespaces;
  @Inject
  Event<CustomResourceDeleted> deletedEvent;
  @Inject
  Event<CustomResourceModified> modifiedEvent;
  @Inject
  Event<CustomResourceAdded> createdEvent;
  @Inject
  Event<CustomResourceOtherInstanceAdded> otherInstanceCreatedEvent;
  @Inject
  FatalErrorHandler fatalErrorHandler;
  @Inject
  NotificationOptions asyncSerial;
  private static final Logger LOGGER = LoggerFactory.getLogger(CustomResourceWatcher.class);

  private final AtomicReference<InstanaAgent> current = new AtomicReference<>();

  public void onElectedLeader(@ObservesAsync OperatorLeaderElected _ev) {
    List<FilterWatchListMultiDeletable<InstanaAgent, InstanaAgentList, Boolean, Watch, Watcher<InstanaAgent>>> ops = new ArrayList<>();
    if (targetNamespaces.isEmpty()) {
      LOGGER.info("Watching for " + KubernetesClientProducer.CRD_NAME + " resources in any namespace.");
      ops.add(client.inAnyNamespace());
    } else {
      for (String targetNamespace : targetNamespaces) {
        LOGGER.info("Watching for " + KubernetesClientProducer.CRD_NAME + " resources in namespace " + targetNamespace);
        ops.add(client.inNamespace(targetNamespace));
      }
    }
    Cache<InstanaAgent, InstanaAgentList> cache = cacheService.newCache(InstanaAgent.class, InstanaAgentList.class);
    for (FilterWatchListMultiDeletable<InstanaAgent, InstanaAgentList, Boolean, Watch, Watcher<InstanaAgent>> op : ops) {
      cache.listThenWatch(op).subscribe(event -> handleCacheEvent(cache.get(event.getUid())));
    }
  }

  // Note that this just forks off the business logic to the cid-handler thread.
  // We don't want to do business logic in the k8s-handler thread.
  private void handleCacheEvent(Optional<InstanaAgent> observed) {
    if (!observed.isPresent()) {
      InstanaAgent previous = current.getAndSet(null);
      if (previous != null) {
        deletedEvent.fireAsync(new CustomResourceDeleted(previous), asyncSerial);
      }
    } else { // observed is present
      InstanaAgent previous = current.get();
      if (previous != null) {
        if (previous.getMetadata().getUid().equals(observed.get().getMetadata().getUid())) {
          // existing instance was modified
          current.set(observed.get());
          modifiedEvent.fireAsync(new CustomResourceModified(previous, observed.get()), asyncSerial)
              .exceptionally(fatalErrorHandler::logAndExit);
        } else {
          // new instance was added. it should be ignored.
          otherInstanceCreatedEvent.fireAsync(new CustomResourceOtherInstanceAdded(previous, observed.get()), asyncSerial)
              .exceptionally(fatalErrorHandler::logAndExit);
        }
      } else { // previous == null
        current.set(observed.get());
        createdEvent.fireAsync(new CustomResourceAdded(observed.get()), asyncSerial)
            .exceptionally(fatalErrorHandler::logAndExit);
      }
    }
  }
}
