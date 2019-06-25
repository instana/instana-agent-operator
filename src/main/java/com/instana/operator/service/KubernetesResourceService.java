package com.instana.operator.service;

import com.instana.operator.GlobalErrorEvent;
import com.instana.operator.kubernetes.Client;
import com.instana.operator.kubernetes.Watchable;
import com.instana.operator.kubernetes.impl.ClientImpl;
import io.fabric8.kubernetes.api.model.*;
import io.fabric8.kubernetes.client.Config;
import io.fabric8.kubernetes.client.DefaultKubernetesClient;
import io.fabric8.kubernetes.client.KubernetesClientException;
import io.quarkus.runtime.ShutdownEvent;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.event.Observes;
import javax.inject.Inject;
import java.util.Map;
import java.util.Optional;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.function.Function;

import static com.instana.operator.util.ConfigUtils.createClientConfig;
import static com.instana.operator.util.DateTimeUtils.nowUTC;
import static com.instana.operator.util.OkHttpClientUtils.createHttpClient;

@ApplicationScoped
public class KubernetesResourceService {

  private static final Logger LOGGER = LoggerFactory.getLogger(KubernetesResourceService.class);

  @Inject
  javax.enterprise.event.Event<GlobalErrorEvent> errorEvent;

  private final Client kubernetesClient;

  private final Map<String, AtomicInteger> countsByType = new ConcurrentHashMap<>();
  private final Map<String, String> firstTimestampByType = new ConcurrentHashMap<>();

  public KubernetesResourceService() throws Exception {
    Config config = createClientConfig();
    this.kubernetesClient = new ClientImpl(new DefaultKubernetesClient(createHttpClient(config), config));
  }

  public Client getKubernetesClient() {
    return kubernetesClient;
  }

  public Optional<Event> sendEvent(String eventName,
                                   String namespace,
                                   String reason,
                                   String message,
                                   String ownerApiVersion,
                                   String ownerKind,
                                   String ownerNamespace,
                                   String ownerName,
                                   String ownerUid) {
    AtomicInteger count = countsByType.computeIfAbsent(eventName, _k -> new AtomicInteger());
    String firstTimestamp = firstTimestampByType.computeIfAbsent(eventName, _k -> nowUTC());

    EventBuilder eb = new EventBuilder()
        .withApiVersion("v1")
        .withNewMetadata()
        .withNamespace(namespace)
        .withGenerateName(eventName + "-")
        .endMetadata()
        .withCount(count.incrementAndGet())
        .withFirstTimestamp(firstTimestamp)
        .withLastTimestamp(nowUTC())
        .withReason(reason)
        .withMessage(message)
        .withType("Normal")
        .withInvolvedObject(new ObjectReferenceBuilder()
            .withApiVersion(ownerApiVersion)
            .withKind(ownerKind)
            .withNamespace(ownerNamespace)
            .withName(ownerName)
            .withUid(ownerUid)
            .build());
    Event event = eb.build();

    try {
      return Optional.ofNullable((Event) kubernetesClient.create(ownerNamespace, event));
    } catch (KubernetesClientException e) {
      if (e.getCause() instanceof java.io.InterruptedIOException) {
        LOGGER.error("Interrupt while creating {} {}/{}.", event.getKind(), event.getMetadata().getNamespace(), event.getMetadata().getName());
      } else {
        LOGGER.error("Could not create {} {}/{}: {}", event.getKind(), event.getMetadata().getNamespace(), event.getMetadata().getName(), e.getMessage(), e);
      }
      return Optional.empty();
    }
  }

  public <T extends HasMetadata, L extends KubernetesResourceList<T>> ResourceCache<T> createResourceCache(
      String name,
      Function<Client, Watchable> fn) {
    Watchable watchable;
    try {
      watchable = fn.apply(kubernetesClient);
    } catch (Throwable t) {
      errorEvent.fire(new GlobalErrorEvent("Failed to watch " + name + ": " + t.getMessage(), t));
      return null; // This line will never be executed, because the error event handler will call System.exit();
    }

    return new ResourceCache<T>(name, watchable, errorEvent);
  }

  void onShutdown(@Observes ShutdownEvent ev) {
    kubernetesClient.close();
  }

}
