package com.instana.operator;

import com.instana.operator.cache.Cache;
import com.instana.operator.cache.CacheService;
import com.instana.operator.customresource.InstanaAgent;
import com.instana.operator.customresource.InstanaAgentSpec;
import com.instana.operator.events.DaemonSetAdded;
import com.instana.operator.events.DaemonSetDeleted;
import io.fabric8.kubernetes.api.model.ConfigMap;
import io.fabric8.kubernetes.api.model.ConfigMapList;
import io.fabric8.kubernetes.api.model.Container;
import io.fabric8.kubernetes.api.model.Doneable;
import io.fabric8.kubernetes.api.model.DoneableConfigMap;
import io.fabric8.kubernetes.api.model.DoneableSecret;
import io.fabric8.kubernetes.api.model.DoneableServiceAccount;
import io.fabric8.kubernetes.api.model.EnvVar;
import io.fabric8.kubernetes.api.model.EnvVarBuilder;
import io.fabric8.kubernetes.api.model.HasMetadata;
import io.fabric8.kubernetes.api.model.HostPathVolumeSource;
import io.fabric8.kubernetes.api.model.HostPathVolumeSourceBuilder;
import io.fabric8.kubernetes.api.model.KubernetesResourceList;
import io.fabric8.kubernetes.api.model.Quantity;
import io.fabric8.kubernetes.api.model.QuantityBuilder;
import io.fabric8.kubernetes.api.model.ResourceRequirements;
import io.fabric8.kubernetes.api.model.Secret;
import io.fabric8.kubernetes.api.model.SecretList;
import io.fabric8.kubernetes.api.model.ServiceAccount;
import io.fabric8.kubernetes.api.model.ServiceAccountList;
import io.fabric8.kubernetes.api.model.Volume;
import io.fabric8.kubernetes.api.model.VolumeBuilder;
import io.fabric8.kubernetes.api.model.VolumeMount;
import io.fabric8.kubernetes.api.model.VolumeMountBuilder;
import io.fabric8.kubernetes.api.model.apps.DaemonSet;
import io.fabric8.kubernetes.api.model.apps.DaemonSetList;
import io.fabric8.kubernetes.api.model.apps.DoneableDaemonSet;
import io.fabric8.kubernetes.api.model.rbac.ClusterRole;
import io.fabric8.kubernetes.api.model.rbac.ClusterRoleBinding;
import io.fabric8.kubernetes.api.model.rbac.ClusterRoleBindingList;
import io.fabric8.kubernetes.api.model.rbac.ClusterRoleList;
import io.fabric8.kubernetes.api.model.rbac.DoneableClusterRole;
import io.fabric8.kubernetes.api.model.rbac.DoneableClusterRoleBinding;
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
import java.util.Base64;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;

import static com.instana.operator.util.ResourceUtils.hasOwner;
import static com.instana.operator.util.ResourceUtils.hasSameName;
import static com.instana.operator.util.ResourceUtils.name;
import static com.instana.operator.util.StringUtils.isBlank;
import static io.fabric8.kubernetes.client.Watcher.Action.ADDED;
import static io.fabric8.kubernetes.client.Watcher.Action.DELETED;

@ApplicationScoped
public class AgentDeployer {

  private static final String DAEMON_SET_NAME = "instana-agent";

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

  // The custom endpoints will be the owner of all resources we create.
  private InstanaAgent owner = null;

  // We will watch all created resources so that we can re-create them if they are deleted.
  private Disposable[] watchers = null;

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
  }

  void customResourceDeleted() {
    if (watchers != null) {
      for (Disposable watcher : watchers) {
        watcher.dispose();
      }
    }
    watchers = null;
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
        }
    );
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
        if (e.getCode() == 409 && nRetries > 1) {
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
    secret.setData(Collections.singletonMap("key", base64(owner.getSpec().getAgentKey())));
    return secret;
  }

  private ConfigMap newConfigMap(InstanaAgent owner,
                                 MixedOperation<ConfigMap, ConfigMapList, DoneableConfigMap, Resource<ConfigMap, DoneableConfigMap>> op) {
    ConfigMap configMap = load("instana-agent.configmap.yaml", owner, op);
    configMap.setData(owner.getSpec().getConfigFiles());
    return configMap;
  }

  private DaemonSet newDaemonSet(InstanaAgent owner,
                                 MixedOperation<DaemonSet, DaemonSetList, DoneableDaemonSet, Resource<DaemonSet, DoneableDaemonSet>> op) {
    InstanaAgentSpec config = owner.getSpec();
    DaemonSet daemonSet = load("instana-agent.daemonset.yaml", owner, op);

    Container container = daemonSet.getSpec().getTemplate().getSpec().getContainers().get(0);
    List<EnvVar> env = container.getEnv();

    env.add(createEnvVar("INSTANA_ZONE", config.getAgentZoneName()));
    env.add(createEnvVar("INSTANA_AGENT_ENDPOINT", config.getAgentEndpointHost()));
    env.add(createEnvVar("INSTANA_AGENT_ENDPOINT_PORT", "" + config.getAgentEndpointPort()));
    env.add(createEnvVar("INSTANA_AGENT_MODE", config.getAgentMode()));
    env.add(createEnvVar("JAVA_OPTS", "-Xmx" + config.getAgentMemLimit() / 3 + "M -XX:+ExitOnOutOfMemoryError"));

    if (!isBlank(config.getAgentDownloadKey())) {
      env.add(createEnvVar("INSTANA_DOWNLOAD_KEY", config.getAgentDownloadKey()));
    }
    if (!isBlank(config.getAgentProxyHost())) {
      env.add(createEnvVar("INSTANA_AGENT_PROXY_HOST", config.getAgentProxyHost()));
    }
    if (config.getAgentProxyPort() != null) {
      env.add(createEnvVar("INSTANA_AGENT_PROXY_PORT", config.getAgentProxyPort().toString()));
    }
    if (!isBlank(config.getAgentProxyProtocol())) {
      env.add(createEnvVar("INSTANA_AGENT_PROXY_PROTOCOL", config.getAgentProxyProtocol()));
    }
    if (!isBlank(config.getAgentProxyUser())) {
      env.add(createEnvVar("INSTANA_AGENT_PROXY_USER", config.getAgentProxyUser()));
    }
    if (!isBlank(config.getAgentProxyPassword())) {
      env.add(createEnvVar("INSTANA_AGENT_PROXY_PASSWORD", config.getAgentProxyPassword()));
    }
    if (config.isAgentProxyUseDNS() != null && config.isAgentProxyUseDNS()) {
      env.add(createEnvVar("INSTANA_AGENT_PROXY_USE_DNS", config.isAgentProxyUseDNS().toString()));
    }
    if (!isBlank(config.getAgentHttpListen())) {
      env.add(createEnvVar("INSTANA_AGENT_HTTP_LISTEN", config.getAgentHttpListen()));
    }
    if (!isBlank(config.getClusterName())) {
      env.add(createEnvVar("INSTANA_KUBERNETES_CLUSTER_NAME", config.getClusterName()));
    }
    System.getenv().entrySet().stream()
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

    return daemonSet;
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
      return resource;
    } catch (Exception e) {
      LOGGER.error("Failed to load " + filename + " from classpath: " + e.getMessage(), e);
      fatalErrorHandler.systemExit(-1);
      return null; // will not happen, because we called System.exit(-1);
    }
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

