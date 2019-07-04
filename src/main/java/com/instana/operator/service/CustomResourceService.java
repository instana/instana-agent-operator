package com.instana.operator.service;

import com.instana.operator.GlobalErrorEvent;
import com.instana.operator.customresource.InstanaAgent;
import com.instana.operator.leaderelection.ElectedLeaderEvent;
import io.fabric8.kubernetes.api.model.apiextensions.CustomResourceDefinition;
import io.fabric8.kubernetes.internal.KubernetesDeserializer;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.event.Event;
import javax.enterprise.event.ObservesAsync;
import javax.inject.Inject;
import java.util.Optional;
import java.util.concurrent.atomic.AtomicReference;

@ApplicationScoped
public class CustomResourceService {

  private static final Logger LOGGER = LoggerFactory.getLogger(CustomResourceService.class);

  private static final String CRD_GROUP = "instana.io";
  private static final String CRD_NAME = "agents." + CRD_GROUP;
  private static final String CRD_VERSION = "v1alpha1";
  private static final String CRD_KIND = InstanaAgent.class.getSimpleName();

  @Inject
  KubernetesResourceService clientService;
  @Inject
  OperatorNamespaceService namespaceService;
  @Inject
  Event<GlobalErrorEvent> globalErrorEvent;
  @Inject
  Event<AgentConfigFoundEvent> agentConfigFoundEvent;

  private final AtomicReference<InstanaAgent> config = new AtomicReference<>();

  void onLeaderElection(@ObservesAsync ElectedLeaderEvent ev) {
    CustomResourceDefinition crd = clientService.getKubernetesClient().get(namespaceService.getNamespace(), CRD_NAME, CustomResourceDefinition.class);
    if (null == crd) {
      globalErrorEvent.fire(new GlobalErrorEvent("Custom resource definition " + CRD_NAME + " not found. Please create the CRD using the provided YAML."));
      return;
    }
    KubernetesDeserializer.registerCustomKind(CRD_GROUP + "/" + CRD_VERSION, CRD_KIND, InstanaAgent.class);
    clientService.getKubernetesClient().registerCustomResource(namespaceService.getNamespace(), crd, InstanaAgent.class);
    ResourceCache<InstanaAgent> resourceCache = clientService.createResourceCache(CRD_NAME, client -> client.watch(namespaceService.getNamespace(), InstanaAgent.class));
    resourceCache.observe()
        .filter(ResourceCache.ChangeEvent::isAdded)
        .subscribe(
            addedEvent -> {
              if (config.compareAndSet(null, addedEvent.getNextValue())) {
                agentConfigFoundEvent.fireAsync(new AgentConfigFoundEvent(addedEvent.getNextValue()));
              }
            },
            throwable -> globalErrorEvent.fire(new GlobalErrorEvent("Failed to watch " + CRD_NAME + "/" + CRD_VERSION + ": " + throwable.getMessage(), throwable))
        );
  }

  public void storeElectedLeaderName(String electedLeaderName) {
    LOGGER.debug("Storing elected leader name " + electedLeaderName);
    config.get().getMetadata().getAnnotations().put("elected-leader", electedLeaderName);
    clientService.getKubernetesClient().createOrUpdate(config.get());
    LOGGER.debug("Done storing elected leader name " + electedLeaderName);
  }

  public Optional<String> loadElectedLeaderName() {
    if (config.get() != null && config.get().getMetadata() != null && config.get().getMetadata().getAnnotations() != null) {
      return Optional.ofNullable(config.get().getMetadata().getAnnotations().get("elected-leader"));
    } else {
      return Optional.empty();
    }
  }

  public void updateStatus(InstanaAgent config) {
    clientService.getKubernetesClient().createOrUpdate(config);
  }
}
