package com.instana.operator;

import com.instana.operator.ExecutorProducer.SingleThreadFactoryBuilder;
import com.instana.operator.cache.Cache;
import com.instana.operator.cache.CacheService;
import com.instana.operator.events.OperatorLeaderElected;
import com.instana.operator.events.OperatorPodValidated;
import io.fabric8.kubernetes.api.model.ConfigMap;
import io.fabric8.kubernetes.api.model.ConfigMapBuilder;
import io.fabric8.kubernetes.api.model.ConfigMapList;
import io.fabric8.kubernetes.api.model.OwnerReference;
import io.fabric8.kubernetes.api.model.OwnerReferenceBuilder;
import io.fabric8.kubernetes.api.model.Pod;
import io.fabric8.kubernetes.client.DefaultKubernetesClient;
import io.fabric8.kubernetes.client.KubernetesClientException;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.event.Event;
import javax.enterprise.event.NotificationOptions;
import javax.enterprise.event.ObservesAsync;
import javax.inject.Inject;
import javax.inject.Named;
import java.util.Optional;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicBoolean;

import static com.instana.operator.env.NamespaceProducer.OPERATOR_NAMESPACE;
import static com.instana.operator.env.OperatorPodNameProducer.POD_NAME;
import static com.instana.operator.util.ResourceUtils.hasOwner;
import static com.instana.operator.util.ResourceUtils.name;

// Java implementation of https://github.com/operator-framework/operator-sdk/blob/master/pkg/leader/leader.go
// If more than one instana-agent-operator Pod is running, this is used to define which of these will be the leader.
@ApplicationScoped
public class OperatorLeaderElector {

  private static final Logger LOGGER = LoggerFactory.getLogger(OperatorLeaderElector.class);

  private static final String LEADER_LOCK = "instana-agent-operator-leader-lock";

  // This will become true if we are the leader, i.e. if the lock with the current
  // Pod as owner reference is present.
  private final AtomicBoolean leader = new AtomicBoolean(false);

  // Indicates if a lock is present. We only try to create a lock if the lock is not present.
  // Initially, this value must be false, because otherwise we will not start trying to
  // create the lock.
  private final AtomicBoolean lockPresent = new AtomicBoolean(false);

  @Inject
  DefaultKubernetesClient client;
  @Inject
  CacheService cacheService;
  @Inject
  Event<OperatorLeaderElected> electedLeaderEvent;
  @Inject
  @Named(OPERATOR_NAMESPACE)
  String operatorNamespace;
  @Inject
  @Named(POD_NAME)
  String podName;
  @Inject
  FatalErrorHandler fatalErrorHandler;
  @Inject
  KubernetesEventService kubernetesEventService;
  @Inject
  SingleThreadFactoryBuilder threadFactoryBuilder;
  @Inject
  NotificationOptions asyncSerial;

  void onOperatorPodValidated(@ObservesAsync OperatorPodValidated event) {
    OwnerReference myself = makeOwnerReference(event.getOperatorPod());
    ScheduledExecutorService executor = Executors.newSingleThreadScheduledExecutor(threadFactoryBuilder.build("try-create-leader-lock"));
    executor.scheduleAtFixedRate(() -> tryCreateLock(myself), 0, 10, TimeUnit.SECONDS);
    watchForLock(myself, () -> asyncSerial.getExecutor().execute(() -> {
      try {
        LOGGER.info("The current Pod " + operatorNamespace + "/" + myself.getName() + " became the leading operator" +
                " instance. Taking over.");
        executor.shutdown();
        executor.awaitTermination(1, TimeUnit.MINUTES);

        createKubernetesEvent(event.getOperatorPod());
        electedLeaderEvent.fireAsync(new OperatorLeaderElected(), asyncSerial)
            .exceptionally(fatalErrorHandler::logAndExit);
      } catch (Exception e) {
        LOGGER.error("Unexpected error while watching ConfigMap " + operatorNamespace + "/" + LEADER_LOCK + ": " + e
            .getMessage(), e);
        fatalErrorHandler.systemExit(-1);
      }
    }));
  }

  private OwnerReference makeOwnerReference(Pod pod) {
    return new OwnerReferenceBuilder()
        .withApiVersion("v1")
        .withKind("Pod")
        .withName(pod.getMetadata().getName())
        .withUid(pod.getMetadata().getUid())
        .build();
  }

  private void tryCreateLock(OwnerReference myself) {
    try {
      if (!lockPresent.get()) {
        ConfigMap lock = new ConfigMapBuilder()
            .withNewMetadata()
            .withName(LEADER_LOCK)
            .withOwnerReferences(myself)
            .endMetadata()
            .build();
        try {
          LOGGER.debug("Trying to create ConfigMap " + operatorNamespace + "/" + LEADER_LOCK + "...");
          client.configMaps().create(lock);
          LOGGER.debug("Successfully created ConfigMap " + operatorNamespace + "/" + LEADER_LOCK + ".");
        } catch (KubernetesClientException e) {
          // For status codes, see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#http-status-codes
          if (e.getCode() == 409) {
            LOGGER.debug("Another operator instance was faster creating ConfigMap " + operatorNamespace + "/" + LEADER_LOCK + ".");
          } else {
            LOGGER.error("Failed to create ConfigMap " + operatorNamespace + "/" + LEADER_LOCK + ": " + e.getMessage(), e);
            fatalErrorHandler.systemExit(-1);
          }
        }
      }
    } catch (Exception e) {
      LOGGER.error("Unexpected error while creating ConfigMap " + operatorNamespace + "/" + LEADER_LOCK + ": " + e.getMessage(), e);
      fatalErrorHandler.systemExit(-1);
    }
  }

  private void watchForLock(OwnerReference myself, Runnable becameLeaderCallback) {
    Cache<ConfigMap, ConfigMapList> cache = cacheService.newCache(ConfigMap.class, ConfigMapList.class);
    cache.listThenWatch(client.inNamespace(operatorNamespace).configMaps().withField("metadata.name", LEADER_LOCK)).subscribe(event -> {
      Optional<ConfigMap> lock = cache.get(event.getUid());
      lockPresent.set(lock.isPresent());
      boolean iOwnTheLock = lock.filter(hasOwner(myself)).isPresent();
      if (iOwnTheLock && !leader.get()) {
        leader.set(true);
        becameLeaderCallback.run();
      }
      if (!iOwnTheLock && leader.get()) {
        LOGGER.error("ConfigMap " + operatorNamespace + "/" + LEADER_LOCK + " disappeared. Terminating.");
        fatalErrorHandler.systemExit(-1);
      }
      if (lock.isPresent() && !iOwnTheLock && !leader.get()) {
        for (OwnerReference owner : lock.get().getMetadata().getOwnerReferences()) {
          LOGGER.info("Another Pod became the leading operator instance: " + owner.getKind() + " " + owner.getName());
        }
      }
    });
  }

  private void createKubernetesEvent(Pod myself) {
    kubernetesEventService.createKubernetesEvent(operatorNamespace, OperatorLeaderElected.class.getSimpleName(),
        "Pod " + name(myself) + " became the leading instana agent operator instance.", myself);
  }
}
