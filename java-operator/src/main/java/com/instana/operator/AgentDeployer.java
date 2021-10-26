/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc.
 */
package com.instana.operator;

import com.instana.operator.cache.Cache;
import com.instana.operator.cache.CacheService;
import com.instana.operator.customresource.InstanaAgent;
import com.instana.operator.customresource.InstanaAgentSpec;
import com.instana.operator.env.Environment;
import com.instana.operator.events.DaemonSetAdded;
import com.instana.operator.events.DaemonSetDeleted;
import io.fabric8.kubernetes.api.model.*;
import io.fabric8.kubernetes.api.model.apiextensions.v1beta1.CustomResourceDefinition;
import io.fabric8.kubernetes.api.model.apps.DaemonSet;
import io.fabric8.kubernetes.api.model.apps.DaemonSetList;
import io.fabric8.kubernetes.api.model.apps.DoneableDaemonSet;
import io.fabric8.kubernetes.api.model.rbac.*;
import io.fabric8.kubernetes.client.DefaultKubernetesClient;
import io.fabric8.kubernetes.client.KubernetesClientException;
import io.fabric8.kubernetes.client.NamespacedKubernetesClient;
import io.fabric8.kubernetes.client.dsl.MixedOperation;
import io.fabric8.kubernetes.client.dsl.Resource;
import io.reactivex.disposables.Disposable;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.event.Event;
import javax.enterprise.event.NotificationOptions;
import javax.inject.Inject;
import javax.inject.Named;
import java.nio.charset.Charset;
import java.util.*;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;

import static com.instana.operator.client.KubernetesClientProducer.CRD_NAME;
import static com.instana.operator.util.ResourceUtils.*;
import static com.instana.operator.util.StringUtils.getBoolean;
import static com.instana.operator.util.StringUtils.isBlank;
import static io.fabric8.kubernetes.client.Watcher.Action.ADDED;
import static io.fabric8.kubernetes.client.Watcher.Action.DELETED;
import static java.net.HttpURLConnection.HTTP_CONFLICT;

@ApplicationScoped
public class AgentDeployer {

  private static final String DEFAULT_NAME_TLS = "instana-agent-tls";
  private static final String DAEMON_SET_NAME = "instana-agent";
  private static final String VERSION_LABEL = "app.kubernetes.io/version";

  @Inject
  DefaultKubernetesClient defaultClient;
  @Inject
  CacheService cacheService;
  @Inject
  FatalErrorHandler fatalErrorHandler;
  @Inject
  ResourceYamlLogger resourceYamlLogger;
  @Inject
  Event<DaemonSetAdded> daemonSetAddedEvent;
  @Inject
  Event<DaemonSetDeleted> daemonSetDeletedEvent;
  @Inject
  NotificationOptions asyncSerial;
  @Inject
  CustomResourceState customResourceState;
  @Inject
  @Named(ExecutorProducer.CDI_HANDLER)
  ScheduledExecutorService executor;
  @Inject
  Environment environment;

  // The custom endpoints will be the owner of all resources we create.
  private InstanaAgent owner = null;

  // We will watch all created resources so that we can re-create them if they are deleted.
  private Disposable[] watchers = null;
  private Disposable tlsSecretWatcher = null;

  private static final Logger LOGGER = LoggerFactory.getLogger(AgentDeployer.class);

  public void customResourceAdded(InstanaAgent customResource) {
    if (owner != null) {
      LOGGER.error("Illegal state: Custom resource " + name(customResource) + " was added, but custom resource " +
          name(owner) + " already exists.");
      fatalErrorHandler.systemExit(-1);
    }
    owner = customResource;
    String targetNamespace = owner.getMetadata().getNamespace();
    NamespacedKubernetesClient c = defaultClient.inNamespace(targetNamespace);

    watchers = new Disposable[] {
        create(ServiceAccount.class, ServiceAccountList.class, this::newServiceAccount, c.serviceAccounts()),
        create(ClusterRole.class, ClusterRoleList.class, this::newClusterRole, c.rbac().clusterRoles()),
        create(ClusterRoleBinding.class, ClusterRoleBindingList.class, this::newCRB, c.rbac().clusterRoleBindings()),
        create(Secret.class, SecretList.class, this::newSecret, c.secrets()),
        create(ConfigMap.class, ConfigMapList.class, this::newConfigMap, c.configMaps()),
        create(DaemonSet.class, DaemonSetList.class, this::newDaemonSet, c.apps().daemonSets()),
        watchDaemonSets(targetNamespace)
    };

    // additional watcher only exists if TLS is configured with certificate and private key
    // if TLS is configured with an existing secret, it does not have to create a new secret
    if (isTlsEncryptionConfigured(owner.getSpec()) && isBlank(owner.getSpec().getAgentTlsSecretName())) {
      tlsSecretWatcher = create(Secret.class, SecretList.class, this::createTlsSecret, c.secrets());
    }
  }

  void customResourceDeleted() {
    if (watchers != null) {
      for (Disposable watcher : watchers) {
        watcher.dispose();
      }
    }
    watchers = null;

    if (tlsSecretWatcher != null) {
      tlsSecretWatcher.dispose();
    }
    tlsSecretWatcher = null;

    owner = null;
    daemonSetDeletedEvent.fireAsync(new DaemonSetDeleted(), asyncSerial)
        .exceptionally(fatalErrorHandler::logAndExit);
  }

  private Disposable watchDaemonSets(String targetNamespace) {
    Cache<DaemonSet, DaemonSetList> daemonSetCache = cacheService.newCache(DaemonSet.class, DaemonSetList.class);
    return daemonSetCache.listThenWatch(
        defaultClient.inNamespace(targetNamespace).apps().daemonSets().withField("metadata.name", DAEMON_SET_NAME))
        .subscribe(
            ev -> {
              if (ev.getAction() == ADDED) {
                Optional<DaemonSet> daemonSet = daemonSetCache.get(ev.getUid())
                    .filter(ds -> hasOwner(ds, owner));
                if (daemonSet.isPresent()) {
                  daemonSetAddedEvent.fireAsync(new DaemonSetAdded(daemonSet.get()), asyncSerial)
                      .exceptionally(fatalErrorHandler::logAndExit);
                }
              }
              if (ev.getAction() == DELETED) {
                // Note that the DELETED event is not reliable, because the watcher might be disposed
                // before the DELETED event is triggered.
                daemonSetDeletedEvent.fireAsync(new DaemonSetDeleted(), asyncSerial)
                    .exceptionally(fatalErrorHandler::logAndExit);
              }
            });
  }

  private <T extends HasMetadata, L extends KubernetesResourceList<T>, D extends Doneable<T>, R extends Resource<T, D>>
  Disposable create(Class<T> resourceClass, Class<L> resourceListClass, Factory<T, L, D, R> factory,
                    MixedOperation<T, L, D, R> op) {
    Cache<T, L> cache = cacheService.newCache(resourceClass, resourceListClass);
    Disposable watch = cache.listThenWatch(op).subscribe(event -> {
          if (!cache.get(event.getUid()).isPresent()) {
            LOGGER.info(resourceClass.getSimpleName() + " has been deleted. Scheduling re-creation.");
            // Delay 5 seconds such that CustomResourceDeleted event is processed before createServiceAccount().
            executor.schedule(() -> createResource(3, op, factory), 5, TimeUnit.SECONDS);
          }
        });
    createResource(3, op, factory);
    return watch;
  }

  private <T extends HasMetadata, L extends KubernetesResourceList<T>, D extends Doneable<T>, R extends Resource<T, D>>
  void createResource(int nRetries, MixedOperation<T, L, D, R> op, Factory<T, L, D, R> factory) {
    if (owner == null) {
      // Custom resource has been deleted.
      return;
    }
    T newResource = null;
    try {
      newResource = factory.newInstance(owner, op);
    } catch (Exception e) {
      LOGGER.error("Failed to generate deployment YAML: " + e.getMessage(), e);
      fatalErrorHandler.systemExit(-1);
    }
    Optional<T> existing = findExisting(newResource, op, owner);
    if (existing.isPresent()) {
      LOGGER.info("Found " + newResource.getKind() + " " + name(newResource) + ".");
      customResourceState.update(existing.get());
    } else {
      try {
        T created = op.create(newResource);
        resourceYamlLogger.log(newResource);
        LOGGER.info("Created " + newResource.getKind() + " " + name(newResource) + ".");
        customResourceState.update(created);
      } catch (KubernetesClientException e) {
        // For status codes, see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#http-status-codes
        if (e.getCode() == HTTP_CONFLICT && nRetries > 1) {
          // Another resource of the same name exists in the same namespace.
          // Maybe it's currently being removed, try again in a few seconds.
          executor.schedule(() -> createResource(nRetries - 1, op, factory), 10, TimeUnit.SECONDS);
        } else {
          resourceYamlLogger.log(newResource);
          LOGGER.error("Failed to create " + newResource.getKind() + " " + name(newResource) + ": "
              + e.getMessage(), e);
          fatalErrorHandler.systemExit(-1);
        }
      }
    }
  }

  private <T extends HasMetadata, L extends KubernetesResourceList<T>, D extends Doneable<T>, R extends Resource<T, D>>
  Optional<T> findExisting(HasMetadata newResource, MixedOperation<T, L, D, R> op, InstanaAgent owner) {
    try {
      return op.list().getItems().stream()
          .filter(existing -> hasSameName(existing, newResource))
          .filter(existing -> hasOwner(existing, owner))
          .findAny();
    } catch (Exception e) {
      LOGGER.error("Failed to list service accounts: " + e.getMessage(), e);
      fatalErrorHandler.systemExit(-1);
      return Optional.empty(); // will not happen, because we called System.exit(-1);
    }
  }

  private ServiceAccount newServiceAccount(InstanaAgent owner,
                                           MixedOperation<ServiceAccount, ServiceAccountList, DoneableServiceAccount, Resource<ServiceAccount, DoneableServiceAccount>> op) {
    return load("instana-agent.serviceaccount.yaml", owner, op);
  }

  private ClusterRole newClusterRole(InstanaAgent owner,
                                     MixedOperation<ClusterRole, ClusterRoleList, DoneableClusterRole, Resource<ClusterRole, DoneableClusterRole>> op) {
    return load("instana-agent.clusterrole.yaml", owner, op);
  }

  private ClusterRoleBinding newCRB(InstanaAgent owner,
                                    MixedOperation<ClusterRoleBinding, ClusterRoleBindingList, DoneableClusterRoleBinding, Resource<ClusterRoleBinding, DoneableClusterRoleBinding>> op) {
    return load("instana-agent.clusterrolebinding.yaml", owner, op);
  }

  private Secret newSecret(InstanaAgent owner,
                           MixedOperation<Secret, SecretList, DoneableSecret, Resource<Secret, DoneableSecret>> op) {
    Secret secret = load("instana-agent.secret.yaml", owner, op);
    HashMap<String, String> secrets = new HashMap<>();
    secrets.put("key", base64(owner.getSpec().getAgentKey()));
    if (!isBlank(owner.getSpec().getAgentDownloadKey())) {
      secrets.put("downloadKey", base64(owner.getSpec().getAgentDownloadKey()));
    }
    secret.setData(secrets);
    return secret;
  }

  private ConfigMap newConfigMap(InstanaAgent owner,
                                 MixedOperation<ConfigMap, ConfigMapList, DoneableConfigMap, Resource<ConfigMap, DoneableConfigMap>> op) {
    ConfigMap configMap = load("instana-agent.configmap.yaml", owner, op);
    final Map<String, String> configFiles = new HashMap<>(owner.getSpec().getConfigFiles());

    // If OpenTelemetry is enabled via Spec, add a ConfigMap entry to enable directly for the Agent
    if (isOpenTelemetryEnabled(owner.getSpec())) {
      configFiles.put("configuration-otel.yaml", "com.instana.plugin.opentelemetry:\n  enabled: true\n");
    }

    configMap.setData(configFiles);

    return configMap;
  }

  DaemonSet newDaemonSet(InstanaAgent owner,
                         MixedOperation<DaemonSet, DaemonSetList, DoneableDaemonSet, Resource<DaemonSet, DoneableDaemonSet>> op) {
    InstanaAgentSpec config = owner.getSpec();
    DaemonSet daemonSet = load("instana-agent.daemonset.yaml", owner, op);

    Container container = daemonSet.getSpec().getTemplate().getSpec().getContainers().get(0);

    String imageFromEnvVar = environment.get(Environment.RELATED_IMAGE_INSTANA_AGENT);
    String imageFromCustomResource = config.getAgentImage();

    if (!isBlank(imageFromCustomResource)) {
      container.setImage(imageFromCustomResource);
    } else if (!isBlank(imageFromEnvVar)) {
      container.setImage(imageFromEnvVar);
    }

    // Get ImagePullPolicy value
    configureAgentImagePullPolicy(config, container);

    // Get and check for OpenTelemetry Settings
    configureOpenTelemetry(container, config);

    List<EnvVar> env = container.getEnv();

    env.add(createEnvVar("INSTANA_ZONE", config.getAgentZoneName()));
    env.add(createEnvVar("INSTANA_AGENT_ENDPOINT", config.getAgentEndpointHost()));
    env.add(createEnvVar("INSTANA_AGENT_ENDPOINT_PORT", "" + config.getAgentEndpointPort()));

    if (!isBlank(config.getAgentDownloadKey())) {
      env.add(new EnvVarBuilder()
          .withName("INSTANA_DOWNLOAD_KEY")
          .withNewValueFrom()
          .withNewSecretKeyRef()
          .withName("instana-agent")
          .withKey("downloadKey")
          .endSecretKeyRef()
          .endValueFrom()
          .build());
    }
    if (!isBlank(config.getClusterName())) {
      env.add(createEnvVar("INSTANA_KUBERNETES_CLUSTER_NAME", config.getClusterName()));
    }
    config.getAgentEnv().forEach((k, v) -> env.add(createEnvVar(k, v)));
    environment.all().entrySet().stream()
        .filter(e -> e.getKey().startsWith("INSTANA_AGENT_") && !"INSTANA_AGENT_KEY".equals(e.getKey()))
        .forEach(e -> {
          env.add(createEnvVar(e.getKey().replaceAll("INSTANA_AGENT_", ""), e.getValue()));
        });

    if (container.getResources() == null) {
      container.setResources(new ResourceRequirements());
    }

    Map<String, Quantity> requests = new HashMap<>();
    requests.put("cpu", cpu(config.getAgentCpuReq()));
    requests.put("memory", mem(config.getAgentMemReq(), "Mi"));
    container.getResources().setRequests(requests);

    Map<String, Quantity> limits = new HashMap<>();
    limits.put("cpu", cpu(config.getAgentCpuLimit()));
    limits.put("memory", mem(config.getAgentMemLimit(), "Mi"));
    container.getResources().setLimits(limits);

    List<VolumeMount> volumeMounts = container.getVolumeMounts();
    owner.getSpec().getConfigFiles().keySet().forEach(fileName ->
        volumeMounts.add(new VolumeMountBuilder()
            .withName("configuration")
            .withMountPath("/root/" + fileName)
            .withSubPath(fileName)
            .build()));

    String hostRepository = owner.getSpec().getAgentHostRepository();
    if (!isBlank(hostRepository)) {
      List<Volume> volumes = daemonSet.getSpec().getTemplate().getSpec().getVolumes();
      volumes.add(new VolumeBuilder()
          .withName("repo")
          .withHostPath(new HostPathVolumeSourceBuilder()
              .withPath(hostRepository)
              .build())
          .build());

      volumeMounts.add(new VolumeMountBuilder()
          .withName("repo")
          .withMountPath("/opt/instana/agent/data/repo")
          .build());
    }

    configureTlsEncryption(container, owner, daemonSet, config);

    return daemonSet;
  }

  private void configureAgentImagePullPolicy(InstanaAgentSpec config, Container container) {
    String imagePullPolicyFromEnvVar = environment.get(Environment.RELATED_IMAGE_PULLPOLICY_INSTANA_AGENT);
    String imagePullPolicyFromCustomResource = config.getAgentImagePullPolicy();

    if (!isBlank(imagePullPolicyFromCustomResource)) {
      container.setImagePullPolicy(imagePullPolicyFromCustomResource);
    } else if (!isBlank(imagePullPolicyFromEnvVar)) {
      container.setImagePullPolicy(imagePullPolicyFromEnvVar);
    }
  }

  private void configureOpenTelemetry(Container container, InstanaAgentSpec config) {
    if (isOpenTelemetryEnabled(config)) {
      container.getPorts().add(new ContainerPort(config.getAgentOtelPort(), null, null, null, null));

      // As we're creating a config-map entry to directly enable OpenTelemetry endpoint, there's also a volume mount needed
      container.getVolumeMounts()
          .add(new VolumeMountBuilder()
              .withName("configuration")
              .withMountPath("/opt/instana/agent/etc/instana/" + "configuration-otel.yaml")
              .withSubPath("configuration-otel.yaml")
              .build());
    }
  }

  private boolean isOpenTelemetryEnabled(InstanaAgentSpec config) {
    String otelActiveFromEnvVar = environment.get(Environment.RELATED_INSTANA_OTEL_ACTIVE);
    Boolean otelActiveFromCustomResource = config.getAgentOpenTelemetryEnabled();

    return otelActiveFromCustomResource || getBoolean(otelActiveFromEnvVar);
  }

  private void configureTlsEncryption(Container container, InstanaAgent owner, DaemonSet daemonSet, InstanaAgentSpec config) {
    if (isTlsEncryptionConfigured(config)) {
      LOGGER.debug("Configure TLS encryption");
      final String secretName = isBlank(config.getAgentTlsSecretName()) ? DEFAULT_NAME_TLS : config.getAgentTlsSecretName();
      final SecretVolumeSource secretVolumeSource = new SecretVolumeSource(0440, new ArrayList<>(), false, secretName);

      daemonSet
          .getSpec()
          .getTemplate()
          .getSpec()
          .getVolumes()
          .add(new VolumeBuilder().withName(DEFAULT_NAME_TLS).withSecret(secretVolumeSource).build());

      container
          .getVolumeMounts()
          .add(new VolumeMountBuilder()
              .withName(DEFAULT_NAME_TLS)
              .withReadOnly(true)
              .withMountPath("/opt/instana/agent/etc/certs")
              .build());
    }
  }

  private boolean isTlsEncryptionConfigured(InstanaAgentSpec config) {
    if (isBlank(config.getAgentTlsSecretName()) && (isBlank(config.getAgentTlsCertificate()) || isBlank(config.getAgentTlsKey()))) {
      return false;
    }
    return true;
  }

  private Secret createTlsSecret(InstanaAgent owner,
                                 MixedOperation<Secret, SecretList, DoneableSecret, Resource<Secret, DoneableSecret>> op) {
    final InstanaAgentSpec config = owner.getSpec();

    LOGGER.debug("Create TLS secret with provided certificate and private key");
    Secret secret = load("instana-agent-tls.secret.yaml", owner, op);
    final Map<String, String> data = new HashMap<String, String>() {{
      put("tls.crt", config.getAgentTlsCertificate());
      put("tls.key", config.getAgentTlsKey());
    }};
    secret.setData(data);
    return secret;

  }

  private Quantity mem(int value, String format) {
    // For some reason the format doesn't work. If we create a Quantity for "512Mi",
    // the resulting YAML contains only "512", which is 512 Bytes.
    // As a workaround, we calculate the value in Bytes.
    if ("Mi".equals(format)) {
      value = value * 1024 * 1024;
    } else {
      throw new IllegalArgumentException("Only format Mi is supported for memory limits.");
    }
    return new QuantityBuilder()
        .withAmount(Integer.toString(value))
        // .withFormat(format)
        .build();
  }

  private Quantity cpu(double value) {
    return new QuantityBuilder()
        .withAmount(Double.toString(value))
        .build();
  }

  private <T extends HasMetadata, L extends KubernetesResourceList<T>, D extends Doneable<T>, R extends Resource<T, D>>
  T load(String filename, InstanaAgent owner, MixedOperation<T, L, D, R> op) {
    try {
      T resource = op.load(getClass().getResourceAsStream("/" + filename)).get();
      resource.getMetadata().setNamespace(owner.getMetadata().getNamespace());
      resource.getMetadata().getOwnerReferences().get(0).setUid(owner.getMetadata().getUid());
      resource.getMetadata().getOwnerReferences().get(0).setName(owner.getMetadata().getName());

      CustomResourceDefinition crd = defaultClient.customResourceDefinitions().withName(CRD_NAME).get();
      if (crd != null && crd.getMetadata().getLabels() != null && crd.getMetadata().getLabels().containsKey(VERSION_LABEL)) {
        resource.getMetadata().getLabels().putIfAbsent(VERSION_LABEL, crd.getMetadata().getLabels().get(VERSION_LABEL));
      }
      return resource;
    } catch (Exception e) {
      LOGGER.error("Failed to load " + filename + " from classpath: " + e.getMessage(), e);
      fatalErrorHandler.systemExit(-1);
      return null; // will not happen, because we called System.exit(-1);
    }
  }

  void setEnvironment(Environment environment) {
    this.environment = environment;
  }

  void setDefaultClient(DefaultKubernetesClient defaultClient) {
    this.defaultClient = defaultClient;
  }

  private interface Factory<T extends HasMetadata, L extends KubernetesResourceList<T>, D extends Doneable<T>, R extends Resource<T, D>> {
    T newInstance(InstanaAgent owner, MixedOperation<T, L, D, R> op);
  }

  private String base64(String secret) {
    return new String(Base64.getEncoder().encode(secret.getBytes(Charset.forName("ASCII"))), Charset.forName("ASCII"));
  }

  private static EnvVar createEnvVar(String name, String value) {
    return new EnvVarBuilder().withName(name).withValue(value).build();
  }
}

