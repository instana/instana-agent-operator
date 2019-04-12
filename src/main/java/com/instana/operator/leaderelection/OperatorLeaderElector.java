package com.instana.operator.leaderelection;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.event.Event;
import javax.enterprise.event.Observes;
import javax.inject.Inject;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

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
  Event<LeaderElectionEvent> leaderElectionEvent;
  @Inject
  Event<GlobalErrorEvent> globalErrorEvent;

  private Disposable watchDisposable;

  private ResourceCache<ConfigMap> defaultConfigMaps;

  void onStartup(@Observes StartupEvent _ev) {
    defaultConfigMaps = clientService.createResourceCache(client ->
        client.configMaps().inNamespace("default"));

    ownerReferenceService.getOperatorPodOwnerReference()
        .thenAccept(ownerRef -> {
          watchDisposable = defaultConfigMaps.observe()
              .filter(changeEvent -> INSTANA_AGENT_OPERATOR_LEADER_LOCK.equals(changeEvent.getName()))
              .subscribe(changeEvent -> {
                boolean areWeLeader;
                if (null == changeEvent.getPreviousValue() || null != changeEvent.getNextValue()) {
                  // ADDED | MODIFIED
                  areWeLeader = changeEvent.getNextValue().getMetadata().getOwnerReferences().stream()
                      .anyMatch(ref -> ref.getUid().equals(ownerRef.getUid()));
                } else {
                  // DELETED
                  areWeLeader = maybeBecomeLeader(ownerRef);
                }

                fireLeaderElectionEvent(areWeLeader, ownerRef);
              });

          if (maybeBecomeLeader(ownerRef)) {
            fireLeaderElectionEvent(true, ownerRef);
          }
        });
  }

  void onShutdown(@Observes ShutdownEvent _ev) {
    if (null != watchDisposable && !watchDisposable.isDisposed()) {
      watchDisposable.dispose();
    }
  }

  private void fireLeaderElectionEvent(boolean areWeLeader, OwnerReference ownerRef) {
    LOGGER.debug("Has {} become leader? {}", ownerRef.getName(), areWeLeader);
    leaderElectionEvent.fire(new LeaderElectionEvent(areWeLeader));

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
      clientService.getKubernetesClient().configMaps().create(cm);
      LOGGER.debug("Successfully created ConfigMap leader lock.");
      return true;
    } catch (KubernetesClientException e) {
      // For status codes, see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#http-status-codes
      if (e.getCode() != 409) {
        globalErrorEvent.fire(new GlobalErrorEvent(e.getCause()));
      }
    }

    return false;
  }

}
