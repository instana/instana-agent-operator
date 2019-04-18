package com.instana.operator.service;

import java.util.Optional;
import java.util.concurrent.CompletableFuture;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.event.Event;
import javax.enterprise.event.Observes;
import javax.inject.Inject;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import com.instana.operator.GlobalErrorEvent;

import io.fabric8.kubernetes.api.model.OwnerReference;
import io.fabric8.kubernetes.api.model.OwnerReferenceBuilder;
import io.fabric8.kubernetes.api.model.Pod;
import io.fabric8.kubernetes.api.model.apps.ReplicaSet;
import io.quarkus.runtime.StartupEvent;
import io.reactivex.disposables.Disposable;

@ApplicationScoped
public class OperatorOwnerReferenceService {

  private static final Logger LOGGER = LoggerFactory.getLogger(OperatorOwnerReferenceService.class);

  @Inject
  KubernetesResourceService clientService;
  @Inject
  OperatorNamespaceService namespaceService;
  @Inject
  Event<GlobalErrorEvent> globalErrorEvent;

  private CompletableFuture<OwnerReference> operatorPodOwnerReference = new CompletableFuture<>();
  private CompletableFuture<OwnerReference> operatorDeploymentOwnerReference = new CompletableFuture<>();

  public CompletableFuture<OwnerReference> getOperatorPodOwnerReference() {
    return operatorPodOwnerReference;
  }

  public CompletableFuture<OwnerReference> getOperatorDeploymentAsOwnerReference() {
    return operatorDeploymentOwnerReference;
  }

  void onStartup(@Observes StartupEvent _ev) {
    ResourceCache<Pod> allPods = clientService.createResourceCache("ownerRef", client -> client.watch(namespaceService.getNamespace(), Pod.class));

    Disposable watch = allPods.observe()
        .filter(changeEvent -> namespaceService.getOperatorPodName().equals(changeEvent.getName()))
        .filter(changeEvent -> !changeEvent.isDeleted())
        .doOnError(t -> globalErrorEvent.fire(new GlobalErrorEvent(t)))
        .subscribe(changeEvent -> {
          operatorPodOwnerReference.complete(new OwnerReferenceBuilder()
              .withApiVersion("v1")
              .withKind("Pod")
              .withName(changeEvent.getNextValue().getMetadata().getName())
              .withUid(changeEvent.getNextValue().getMetadata().getUid())
              .build());

          OwnerReference ref = findReplicaSetOwnerReference(changeEvent.getNextValue())
              .flatMap(this::findReplicaSet)
              .flatMap(this::findDeploymentOwnerReference)
              .orElseThrow(() -> new IllegalStateException("Could not find Operator Pod OwnerReference!"));
          operatorDeploymentOwnerReference.complete(ref);
        });

    operatorPodOwnerReference.whenComplete((pr, t1) -> {
      operatorDeploymentOwnerReference.whenComplete((dr, t2) -> {
        LOGGER.debug("Disposing of the OwnerReference watches...");
        watch.dispose();
      });
    });
  }

  private Optional<OwnerReference> findReplicaSetOwnerReference(Pod pod) {
    return pod.getMetadata().getOwnerReferences().stream()
        .filter(ref -> "ReplicaSet".equals(ref.getKind()))
        .findFirst();
  }

  private Optional<OwnerReference> findDeploymentOwnerReference(ReplicaSet rs) {
    return rs.getMetadata().getOwnerReferences().stream()
        .filter((ref -> "Deployment".equals(ref.getKind())))
        .findFirst();
  }

  private Optional<ReplicaSet> findReplicaSet(OwnerReference ownerRef) {
    return Optional.ofNullable(clientService.getKubernetesClient().get(namespaceService.getNamespace(), ownerRef.getName(), ReplicaSet.class));
  }

}
