package com.instana.operator.leaderelection;

import java.util.Objects;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicBoolean;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.event.Observes;
import javax.inject.Inject;

import org.eclipse.microprofile.config.inject.ConfigProperty;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import io.fabric8.kubernetes.api.model.ConfigMap;
import io.fabric8.kubernetes.api.model.ConfigMapBuilder;
import io.fabric8.kubernetes.api.model.ContainerStatus;
import io.fabric8.kubernetes.api.model.OwnerReference;
import io.fabric8.kubernetes.api.model.OwnerReferenceBuilder;
import io.fabric8.kubernetes.api.model.Pod;
import io.fabric8.kubernetes.client.KubernetesClientException;
import io.fabric8.kubernetes.client.NamespacedKubernetesClient;
import io.fabric8.kubernetes.client.Watch;
import io.fabric8.kubernetes.client.Watcher;
import io.quarkus.runtime.ShutdownEvent;
import io.quarkus.runtime.StartupEvent;
import io.reactivex.Observable;
import io.reactivex.Single;
import io.reactivex.disposables.Disposable;

/**
 * Java implementation of https://github.com/operator-framework/operator-sdk/blob/master/pkg/leader/leader.go
 * If more than one instana-operator Pod is running, this is used to define which of these will be the leader.
 */
@ApplicationScoped
public class ConfigMapLeaderElectorService {

  private static final Logger LOGGER = LoggerFactory.getLogger(ConfigMapLeaderElectorService.class);

  static final String CONFIGMAP_NAME = "instana-operator-leader-elector";

  @ConfigProperty(name = "instana.leaderelection.pollInterval", defaultValue = "10")
  String pollInterval;

  @Inject
  NamespacedKubernetesClient kubernetesClient;

  private static final String POD_NAME = System.getenv("POD_NAME");
  private static final String POD_NAMESPACE = System.getenv("POD_NAMESPACE");

  private OwnerReference ownerReference;

  private final AtomicBoolean isLeader = new AtomicBoolean(false);

  private Watch watch;
  private Disposable poller;

  public ConfigMapLeaderElectorService() {
  }

  void startup(@Observes StartupEvent ev) {
    this.watch = kubernetesClient.pods().inNamespace(POD_NAMESPACE).watch(new Watcher<Pod>() {
      @Override
      public void eventReceived(Action action, Pod resource) {
        if (!resource.getMetadata().getName().equals(POD_NAME)) {
          return;
        }
        LOGGER.debug("MODIFIED {} {}", POD_NAME, resource);
        switch (action) {
        case MODIFIED:
          resource.getStatus().getContainerStatuses().stream()
              .filter(cs -> cs.getImage().contains("operator"))
              .findFirst()
              .map(ContainerStatus::getReady)
              .filter(ready -> ready)
              .ifPresent(ready -> {
                ownerReference = findMyOwnerRef();
                isLeader.set(wasElectedLeader().blockingGet());
                if (null != watch) {
                  watch.close();
                }
              });
          break;
        }
      }

      @Override
      public void onClose(KubernetesClientException cause) {

      }
    });

    long pollIntervalMs = TimeUnit.SECONDS.toMillis(Long.valueOf(pollInterval));
    this.poller = Observable
        .interval(pollIntervalMs, pollIntervalMs, TimeUnit.MILLISECONDS)
        .subscribe(now -> createConfigMap().isEmpty().subscribe(empty -> isLeader.set(!empty)));
  }

  void shutdown(@Observes ShutdownEvent ev) {
    if (null != poller && !poller.isDisposed()) {
      poller.dispose();
    }
  }

  public boolean isLeader() {
    return isLeader.get();
  }

  private Single<Boolean> wasElectedLeader() {
    ConfigMap cm = kubernetesClient.configMaps().withName(CONFIGMAP_NAME).get();
    if (null == cm) {
      // No leader is elected.
      return Single.just(Boolean.FALSE);
    }

    // From OwnerReferences, determine if any were ours (#therecanonlybeone).
    return Observable.fromIterable(cm.getMetadata().getOwnerReferences())
        .filter(ref -> ref.getUid().equals(ownerReference.getUid()))
        .isEmpty()
        .map(not -> !not);
  }

  /*
   * Try to create the ConfigMap with our OwnerReference. If we can, that means we're the leader.
   */
  private Observable<ConfigMap> createConfigMap() {
    if (LOGGER.isDebugEnabled()) {
      LOGGER.debug("Trying to create ConfigMap {} for owner {}/{}", CONFIGMAP_NAME, POD_NAMESPACE, POD_NAME);
    }
    return Observable.just(ownerReference)
        .map(ref -> new ConfigMapBuilder()
            .withNewMetadata()
            .withName(CONFIGMAP_NAME)
            .withOwnerReferences(ref)
            .endMetadata()
            .build())
        .flatMap(cm -> {
          try {
            return Observable.just(kubernetesClient.configMaps().create(cm))
                .doOnNext(_cm -> LOGGER.debug("Created ConfigMap {}", _cm));
          } catch (KubernetesClientException e) {
            if (LOGGER.isDebugEnabled()) {
              LOGGER.debug(e.getMessage(), e);
            }
            // For status codes, see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#http-status-codes
            if (e.getCode() != 409) {
              return Observable.error(e);
            } else {
              return Observable.empty();
            }
          }
        });
  }

  private OwnerReference findMyOwnerRef() {
    Pod self = kubernetesClient.pods().inNamespace(POD_NAMESPACE).withName(POD_NAME).get();
    Objects.requireNonNull(self, "Could not find Operator Pod.");
    return new OwnerReferenceBuilder()
        .withApiVersion("v1")
        .withKind("Pod")
        .withName(self.getMetadata().getName())
        .withUid(self.getMetadata().getUid())
        .build();
  }

}
