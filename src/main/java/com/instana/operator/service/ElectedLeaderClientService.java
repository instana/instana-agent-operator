package com.instana.operator.service;

import java.util.Optional;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.TimeUnit;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.event.Event;
import javax.enterprise.event.Observes;
import javax.inject.Inject;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import com.instana.operator.GlobalErrorEvent;
import com.instana.operator.customresource.DoneableElectedLeader;
import com.instana.operator.customresource.ElectedLeader;
import com.instana.operator.customresource.ElectedLeaderList;
import com.instana.operator.customresource.ElectedLeaderSpec;

import io.fabric8.kubernetes.api.model.ObjectMeta;
import io.fabric8.kubernetes.api.model.ObjectMetaBuilder;
import io.fabric8.kubernetes.api.model.apiextensions.CustomResourceDefinition;
import io.fabric8.kubernetes.api.model.apiextensions.CustomResourceDefinitionList;
import io.fabric8.kubernetes.client.dsl.NonNamespaceOperation;
import io.fabric8.kubernetes.client.dsl.Resource;
import io.fabric8.kubernetes.internal.KubernetesDeserializer;
import io.quarkus.runtime.StartupEvent;

/**
 * Client service to access the custom resource electedleaders.agent.instana.io.
 * <p>
 * The Custom resource definition is created with the following yaml:
 *
 * <pre>
 * apiVersion: apiextensions.k8s.io/v1beta1
 * kind: CustomResourceDefinition
 * metadata:
 *   name: electedleaders.agent.instana.io
 *   namespace: instana-agent
 * spec:
 *   group: agent.instana.io
 *   versions:
 *     - name: v1
 *       served: true
 *       storage: true
 *   scope: Namespaced
 *   names:
 *     plural: electedleaders
 *     singular: electedleader
 *     kind: ElectedLeader
 *     shortNames:
 *     - el
 * </pre>
 * <p>
 * With <tt>kubectl</tt> you can get the current value as follows:
 *
 * <pre>
 * kubectl -n instana-agent get electedleaders.agent.instana.io elected-leader -o yaml
 * </pre>
 */
@ApplicationScoped
public class ElectedLeaderClientService {

  private static final Logger LOGGER = LoggerFactory.getLogger(ElectedLeaderClientService.class);

  private static final String CRD_GROUP = "agent.instana.io";
  private static final String CRD_NAME = "electedleaders." + CRD_GROUP;
  private static final String CRD_VERSION = "v1beta1";
  private static final String CRD_KIND = ElectedLeader.class.getSimpleName();
  private static final String CR_NAME = "elected-leader";

  @Inject
  KubernetesResourceService clientService;
  @Inject
  OperatorNamespaceService namespaceService;
  @Inject
  Event<GlobalErrorEvent> globalErrorEvent;

  private final CompletableFuture<NonNamespaceOperation<ElectedLeader, ElectedLeaderList, DoneableElectedLeader, Resource<ElectedLeader, DoneableElectedLeader>>> client = new CompletableFuture<>();

  void onStartup(@Observes StartupEvent _ev) {
    String namespace = namespaceService.getNamespace();
    CustomResourceDefinitionList crdList = clientService.getKubernetesClient()
        .inNamespace(namespace)
        .customResourceDefinitions()
        .list();
    if (null == crdList || crdList.getItems().isEmpty()) {
      globalErrorEvent.fire(new GlobalErrorEvent(new IllegalStateException(
          "No CustomResourceDefinitions found! Please create the Instana ElectedLeader CRD using the provided YAML.")));
      return;
    }

    Optional<CustomResourceDefinition> crd = crdList.getItems().stream()
        .filter(c -> c.getMetadata() != null && CRD_NAME.equals(c.getMetadata().getName()))
        .findAny();
    if (!crd.isPresent()) {
      globalErrorEvent.fire(new GlobalErrorEvent(
          new IllegalStateException("Custom resource definition " + namespace + "/" + CRD_NAME
              + " not found! Please create the Instana ElectedLeader CRD using the provided YAML.")));
    }

    crd.ifPresent(c -> {
      KubernetesDeserializer.registerCustomKind(CRD_GROUP + "/" + CRD_VERSION, CRD_KIND, ElectedLeader.class);
      client.complete(clientService.getKubernetesClient()
          .customResources(c, ElectedLeader.class, ElectedLeaderList.class, DoneableElectedLeader.class)
          .inNamespace(namespace));
    });
  }

  /**
   * Load the current value.
   */
  public Optional<ElectedLeaderSpec> loadElectedLeader() {
    ElectedLeaderList list = getClient().list();
    if (list == null) {
      LOGGER.debug("ElectedLeader CustomResource was not present.");
      return Optional.empty();
    }
    return list.getItems().stream()
        .filter(el -> el.getMetadata() != null && CR_NAME.equals(el.getMetadata().getName()))
        .map(ElectedLeader::getSpec)
        .peek(s -> LOGGER.debug("ElectedLeader CustomResource was found {}", s))
        .findAny();
  }

  /**
   * Create or replace the current value.
   */
  public ElectedLeader upsertElectedLeader(ElectedLeaderSpec spec) {
    ElectedLeader electedLeader = new ElectedLeader();
    ObjectMeta metadata = new ObjectMetaBuilder()
        .withName(CR_NAME)
        .build();

    electedLeader.setMetadata(metadata);
    electedLeader.setSpec(spec);

    return getClient().createOrReplace(electedLeader);
  }

  /**
   * Use this to call getClient().watch(...);
   */
  public NonNamespaceOperation<ElectedLeader, ElectedLeaderList, DoneableElectedLeader, Resource<ElectedLeader, DoneableElectedLeader>> getClient() {
    try {
      return client.get(2, TimeUnit.MINUTES);
    } catch (Exception e) {
      globalErrorEvent.fire(new GlobalErrorEvent(e));
      return null; // NPE is intentional. The container will shut down anyway.
    }
  }

}
