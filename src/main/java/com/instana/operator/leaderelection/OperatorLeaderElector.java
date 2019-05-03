package com.instana.operator.leaderelection;

import com.instana.operator.GlobalErrorEvent;
import com.instana.operator.service.KubernetesResourceService;
import com.instana.operator.service.OperatorNamespaceService;
import com.instana.operator.service.OperatorOwnerReferenceService;
import com.instana.operator.service.ResourceCache;
import io.fabric8.kubernetes.api.model.ConfigMap;
import io.fabric8.kubernetes.api.model.ConfigMapBuilder;
import io.fabric8.kubernetes.api.model.OwnerReference;
import io.fabric8.kubernetes.client.KubernetesClientException;
import io.quarkus.runtime.ShutdownEvent;
import io.quarkus.runtime.StartupEvent;
import io.reactivex.disposables.Disposable;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.event.Event;
import javax.enterprise.event.Observes;
import javax.inject.Inject;
import java.util.concurrent.ScheduledExecutorService;

import static javax.enterprise.event.NotificationOptions.ofExecutor;

// Java implementation of https://github.com/operator-framework/operator-sdk/blob/master/pkg/leader/leader.go
// If more than one instana-agent-operator Pod is running, this is used to define which of these will be the leader.
@ApplicationScoped
public class OperatorLeaderElector {

  private static final Logger LOGGER = LoggerFactory.getLogger(OperatorLeaderElector.class);

  private static final String INSTANA_AGENT_OPERATOR_LEADER_LOCK = "instana-agent-operator-leader-lock";

  @Inject
  KubernetesResourceService clientService;
  @Inject
  OperatorNamespaceService namespaceService;
  @Inject
  OperatorOwnerReferenceService ownerReferenceService;
  @Inject
  Event<ElectedLeaderEvent> electedLeaderEvent;
  @Inject
  Event<ImpeachedLeaderEvent> impeachedLeaderEvent;
  @Inject
  ScheduledExecutorService executorService;
  @Inject
  Event<GlobalErrorEvent> globalErrorEvent;

  private Disposable watchDisposable;

  private ResourceCache<ConfigMap> defaultConfigMaps;

  private volatile boolean areWeLeader = false;

  void onStartup(@Observes StartupEvent _ev) {
    defaultConfigMaps = clientService.createResourceCache(namespaceService.getNamespace() + "-configMaps", client ->
        client.watch(namespaceService.getNamespace(), ConfigMap.class));

    ownerReferenceService.getOperatorPodOwnerReference()
        .thenAccept(ownerRef -> {
          watchDisposable = defaultConfigMaps.observe()
              .filter(changeEvent -> INSTANA_AGENT_OPERATOR_LEADER_LOCK.equals(changeEvent.getName()))
              .filter(ResourceCache.ChangeEvent::isDeleted)
              .subscribe(changeEvent -> {
                if (!areWeLeader && maybeBecomeLeader(ownerRef)) {
                  areWeLeader = true;
                  fireLeaderElectionEvent(areWeLeader, ownerRef);
                }
              });

          areWeLeader = maybeBecomeLeader(ownerRef);
          fireLeaderElectionEvent(areWeLeader, ownerRef);
        });
  }

  public boolean isThisPodLeader() {
    return areWeLeader;
  }

  void onShutdown(@Observes ShutdownEvent _ev) {
    if (null != watchDisposable && !watchDisposable.isDisposed()) {
      watchDisposable.dispose();
    }
  }

  private void fireLeaderElectionEvent(boolean areWeLeader, OwnerReference ownerRef) {
    if (areWeLeader) {
      electedLeaderEvent.fireAsync(new ElectedLeaderEvent(), ofExecutor(executorService));
    } else if (this.areWeLeader) {
      impeachedLeaderEvent.fireAsync(new ImpeachedLeaderEvent(), ofExecutor(executorService));
    }

    clientService.sendEvent(
        "operator-leader-elected",
        namespaceService.getNamespace(),
        "ElectedLeader",
        "Successfully elected leader: " + namespaceService.getNamespace() + "/" + ownerRef.getName(),
        ownerRef.getApiVersion(),
        ownerRef.getKind(),
        namespaceService.getNamespace(),
        ownerRef.getName(),
        ownerRef.getUid());
  }

  private boolean maybeBecomeLeader(OwnerReference ownerReference) {
    ConfigMap cm = new ConfigMapBuilder()
        .withNewMetadata()
        .withName(INSTANA_AGENT_OPERATOR_LEADER_LOCK)
        .withOwnerReferences(ownerReference)
        .endMetadata()
        .build();

    try {
      LOGGER.debug("Trying to create ConfigMap leader lock...");
      clientService.getKubernetesClient().create(namespaceService.getNamespace(), cm);
      LOGGER.debug("Successfully created ConfigMap leader lock.");
      return true;
    } catch (KubernetesClientException e) {
      // For status codes, see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#http-status-codes
      if (e.getCode() != 409) {
        if (e.getCause() instanceof java.io.InterruptedIOException) {
          LOGGER.error("Thread interrupted while creating config map " + INSTANA_AGENT_OPERATOR_LEADER_LOCK + " in namespace " + namespaceService.getNamespace() + ".");
        } else {
          globalErrorEvent.fire(new GlobalErrorEvent("Received unexpected error code " + e.getCode()
              + " from the Kubernetes API server while creating config map " + INSTANA_AGENT_OPERATOR_LEADER_LOCK
              + " in namespace " + namespaceService.getNamespace() + ".", e));
        }
      }
    }

    return false;
  }

}
