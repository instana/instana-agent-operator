package com.instana.operator.util;

import static java.util.Arrays.asList;
import static org.apache.commons.lang3.StringUtils.isBlank;

import java.util.ArrayList;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import io.fabric8.kubernetes.api.model.ConfigMap;
import io.fabric8.kubernetes.api.model.ConfigMapBuilder;
import io.fabric8.kubernetes.api.model.ConfigMapVolumeSourceBuilder;
import io.fabric8.kubernetes.api.model.ContainerBuilder;
import io.fabric8.kubernetes.api.model.ContainerPortBuilder;
import io.fabric8.kubernetes.api.model.EnvVar;
import io.fabric8.kubernetes.api.model.EnvVarBuilder;
import io.fabric8.kubernetes.api.model.EnvVarSourceBuilder;
import io.fabric8.kubernetes.api.model.HostPathVolumeSourceBuilder;
import io.fabric8.kubernetes.api.model.OwnerReference;
import io.fabric8.kubernetes.api.model.ProbeBuilder;
import io.fabric8.kubernetes.api.model.Quantity;
import io.fabric8.kubernetes.api.model.QuantityBuilder;
import io.fabric8.kubernetes.api.model.Secret;
import io.fabric8.kubernetes.api.model.SecretBuilder;
import io.fabric8.kubernetes.api.model.SecurityContextBuilder;
import io.fabric8.kubernetes.api.model.ServiceAccount;
import io.fabric8.kubernetes.api.model.ServiceAccountBuilder;
import io.fabric8.kubernetes.api.model.Volume;
import io.fabric8.kubernetes.api.model.VolumeBuilder;
import io.fabric8.kubernetes.api.model.VolumeMount;
import io.fabric8.kubernetes.api.model.VolumeMountBuilder;
import io.fabric8.kubernetes.api.model.apps.DaemonSet;
import io.fabric8.kubernetes.api.model.apps.DaemonSetBuilder;
import io.fabric8.kubernetes.api.model.rbac.ClusterRole;
import io.fabric8.kubernetes.api.model.rbac.ClusterRoleBinding;
import io.fabric8.kubernetes.api.model.rbac.ClusterRoleBindingBuilder;
import io.fabric8.kubernetes.api.model.rbac.ClusterRoleBuilder;
import io.fabric8.kubernetes.api.model.rbac.PolicyRule;
import io.fabric8.kubernetes.api.model.rbac.PolicyRuleBuilder;
import io.fabric8.kubernetes.api.model.rbac.SubjectBuilder;

public abstract class AgentResourcesUtil {

  private AgentResourcesUtil() {
  }

  public static ServiceAccount createServiceAccount(String namespace,
                                                    String name,
                                                    OwnerReference ownerReference) {
    return new ServiceAccountBuilder()
        .withNewMetadata()
        .withNamespace(namespace)
        .withName(name)
        .withOwnerReferences(ownerReference)
        .endMetadata()
        .build();
  }

  public static ConfigMap createConfigurationConfigMap(String namespace,
                                                       String name,
                                                       OwnerReference ownerReference) {

    Map<String, String> data = new HashMap<>();
    data.put("configuration.yaml", "|\n");
    data.put("org.ops4j.pax.url.mvn.cfg", "|\n" +
        "org.ops4j.pax.url.mvn.certificateCheck=true\n" +
        "org.ops4j.pax.url.mvn.socket.keepAlive=true\n" +
        "org.ops4j.pax.url.mvn.settings=file:${karaf.etc}/mvn-settings.xml\n" +
        "org.ops4j.pax.url.mvn.useFallbackRepositories=false\n" +
        "org.ops4j.pax.url.mvn.defaultRepositories=file:/root/.m2/repository@id=m2repo@snapshots,file:${karaf.home}/${karaf.default.repository}@id=system.repository@snapshots\n"
        +
        "org.ops4j.pax.url.mvn.repositories=https://artifact-public.instana.io/artifactory/features-public@id=features@noreleases@snapshots@snapshotsUpdate=always,https://artifact-public.instana.io/artifactory/shared@id=shared@snapshots@snapshotsUpdate=always\n");

    return new ConfigMapBuilder()
        .withNewMetadata()
        .withNamespace(namespace)
        .withName(name)
        .withOwnerReferences(ownerReference)
        .endMetadata()
        .withData(data)
        .build();
  }

  public static Secret createAgentKeySecret(String namespace,
                                            String name,
                                            String key,
                                            OwnerReference ownerReference) {
    return new SecretBuilder()
        .withNewMetadata()
        .withNamespace(namespace)
        .withName(name)
        .withOwnerReferences(ownerReference)
        .endMetadata()
        .withData(Collections.singletonMap("key", key))
        .build();
  }

  public static ClusterRole createAgentClusterRole(String name, OwnerReference ownerReference) {
    List<PolicyRule> rules = new ArrayList<>();
    rules.add(createPolicyRule(
        asList("batch"),
        asList("jobs"),
        asList("get", "list", "watch")));
    rules.add(createPolicyRule(
        asList("extensions"),
        asList("deployments",
            "replicasets",
            "ingresses"),
        asList("get", "list", "watch")));
    rules.add(createPolicyRule(
        asList(""),
        asList("namespaces",
            "events",
            "services",
            "endpoints",
            "nodes",
            "pods",
            "replicationcontrollers",
            "componentstatuses",
            "resourcequotas"),
        asList("get", "list", "watch")));
    rules.add(createPolicyRule(
        asList(""),
        asList("endpoints"),
        asList("create", "update")));
    rules.add(createNonResourceURLsPolicyRule(
        asList("/version", "/healthz"),
        asList("get")));

    return new ClusterRoleBuilder()
        .withNewMetadata()
        .withName(name)
        .withOwnerReferences(ownerReference)
        .endMetadata()
        .withRules(rules)
        .build();
  }

  public static ClusterRoleBinding createAgentClusterRoleBinding(String namespace,
                                                                 String name,
                                                                 ServiceAccount serviceAccount,
                                                                 ClusterRole clusterRole,
                                                                 OwnerReference ownerReference) {
    return new ClusterRoleBindingBuilder()
        .withNewMetadata()
        .withNamespace(namespace)
        .withName(name)
        .withOwnerReferences(ownerReference)
        .endMetadata()
        .withSubjects(new SubjectBuilder()
            .withNamespace(namespace)
            .withName(serviceAccount.getMetadata().getName())
            .withKind("ServiceAccount")
            .build())
        .withNewRoleRef()
        .withKind("ClusterRole")
        .withName(clusterRole.getMetadata().getName())
        .endRoleRef()
        .build();
  }

  public static DaemonSet createAgentDaemonSet(String namespace,
                                               String name,
                                               ServiceAccount serviceAccount,
                                               Secret secret,
                                               ConfigMap configMap,
                                               OwnerReference ownerReference,
                                               String downloadKey,
                                               String zone,
                                               String endpoint,
                                               String endpointPort,
                                               String mode,
                                               double cpuReq,
                                               int memoryReq,
                                               double cpuLimit,
                                               int memoryLimit,
                                               String imageName,
                                               String imageTag,
                                               String proxyHost,
                                               String proxyPort,
                                               String proxyProtocol,
                                               String proxyUser,
                                               String proxyPasswd,
                                               String proxyUseDNS,
                                               String httpListen) {
    List<EnvVar> env = new ArrayList<>();
    env.add(createEnvVar("INSTANA_OPERATOR_MANAGED", "true"));
    env.add(createEnvVar("INSTANA_ZONE", zone));
    env.add(createEnvVar("INSTANA_AGENT_ENDPOINT", endpoint));
    env.add(createEnvVar("INSTANA_AGENT_ENDPOINT_PORT", endpointPort));
    env.add(createEnvVar("INSTANA_AGENT_MODE", mode));
    env.add(createEnvVarFromSecret("INSTANA_AGENT_KEY", secret.getMetadata().getName()));
    env.add(createEnvVar("JAVA_OPTS", String.format("-Xmx%dM -XX:+ExitOnOutOfMemoryError", memoryReq / 3)));
    env.add(createEnvVarFromFieldRef("INSTANA_AGENT_POD_NAME", "metadata.name"));

    if (!isBlank(downloadKey)) {
      env.add(createEnvVar("INSTANA_DOWNLOAD_KEY", downloadKey));
    }

    if (!isBlank(proxyHost)) {
      env.add(createEnvVar("INSTANA_AGENT_PROXY_HOST", proxyHost));
    }
    if (!isBlank(proxyPort)) {
      env.add(createEnvVar("INSTANA_AGENT_PROXY_PORT", proxyPort));
    }
    if (!isBlank(proxyProtocol)) {
      env.add(createEnvVar("INSTANA_AGENT_PROXY_PROTOCOL", proxyProtocol));
    }
    if (!isBlank(proxyUser)) {
      env.add(createEnvVar("INSTANA_AGENT_PROXY_USER", proxyUser));
    }
    if (!isBlank(proxyPasswd)) {
      env.add(createEnvVar("INSTANA_AGENT_PROXY_PASSWORD", proxyPasswd));
    }
    if (!isBlank(proxyUseDNS)) {
      env.add(createEnvVar("INSTANA_AGENT_PROXY_USE_DNS", proxyUseDNS));
    }

    if (!isBlank(httpListen)) {
      env.add(createEnvVar("INSTANA_AGENT_HTTP_LISTEN", httpListen));
    }

    System.getenv().entrySet().stream()
        .filter(e -> e.getKey().startsWith("INSTANA_AGENT_") && !"INSTANA_AGENT_KEY".equals(e.getKey()))
        .forEach(e -> {
          env.add(createEnvVar(e.getKey().substring(14), e.getValue()));
        });

    List<VolumeMount> mounts = new ArrayList<>();
    mounts.add(createVolumeMount("dev", "/dev"));
    mounts.add(createVolumeMount("run", "/var/run/docker.sock"));
    mounts.add(createVolumeMount("sys", "/sys"));
    mounts.add(createVolumeMount("log", "/var/log"));
    mounts.add(createVolumeMount("machine-id", "/etc/machine-id"));
    for (String key : configMap.getData().keySet()) {
      mounts.add(createVolumeMount("configuration", "/root/" + key, key));
    }

    List<Volume> vols = new ArrayList<>();
    vols.add(createVolumeFromHostPath("dev", "/dev"));
    vols.add(createVolumeFromHostPath("run", "/var/run/docker.sock"));
    vols.add(createVolumeFromHostPath("sys", "/sys"));
    vols.add(createVolumeFromHostPath("log", "/var/log"));
    vols.add(createVolumeFromHostPath("machine-id", "/etc/machine-id"));
    vols.add(createVolumeFromConfigMap("configuration", configMap.getMetadata().getName()));

    Map<String, Quantity> requests = new HashMap<>();
    requests.put("cpu", createQuantity(cpuReq, null));
    requests.put("memory", createQuantity(memoryReq, "Mi"));

    Map<String, Quantity> limits = new HashMap<>();
    requests.put("cpu", createQuantity(cpuLimit, null));
    requests.put("memory", createQuantity(memoryLimit, "Mi"));

    Map<String, String> labels = new HashMap<>();
    labels.put("agent.instana.io/role", "agent");
    labels.put("app", name);

    return new DaemonSetBuilder()
        .withNewMetadata()
        .withNamespace(namespace)
        .withName(name)
        .withOwnerReferences(ownerReference)
        .endMetadata()
        .withNewSpec()
        .withNewSelector()
        .withMatchLabels(labels)
        .endSelector()
        .withNewTemplate()
        .withNewMetadata()
        .withLabels(labels)
        .endMetadata()
        .withNewSpec()
        .withHostIPC(true)
        .withHostNetwork(true)
        .withHostPID(true)
        .withServiceAccountName(serviceAccount.getMetadata().getName())
        .withContainers(new ContainerBuilder()
            .withName(name)
            .withImage(imageName + ":" + imageTag)
            .withImagePullPolicy("IfNotPresent")
            .withEnv(env)
            .withSecurityContext(new SecurityContextBuilder()
                .withPrivileged(true)
                .build())
            .withVolumeMounts(mounts)
            .withNewResources()
            .withRequests(requests)
            .withLimits(limits)
            .endResources()
            .withLivenessProbe(new ProbeBuilder()
                .withNewHttpGet()
                .withNewPort(42699)
                .withPath("/status")
                .endHttpGet()
                .withInitialDelaySeconds(75)
                .withPeriodSeconds(5)
                .build())
            .withPorts(new ContainerPortBuilder()
                .withContainerPort(42699)
                .build())
            .build())
        .withVolumes(vols)
        .endSpec()
        .endTemplate()
        .endSpec()
        .build();
  }

  public static EnvVar createEnvVar(String name, String value) {
    return new EnvVarBuilder().withName(name).withValue(value).build();
  }

  public static EnvVar createEnvVarFromSecret(String name, String key) {
    return new EnvVarBuilder()
        .withName(name)
        .withValueFrom(new EnvVarSourceBuilder()
            .editOrNewSecretKeyRef()
            .withName(key)
            .withKey("key")
            .endSecretKeyRef()
            .build())
        .build();
  }

  public static EnvVar createEnvVarFromFieldRef(String name, String fieldPath) {
    return new EnvVarBuilder()
        .withName(name)
        .withNewValueFrom()
        .withNewFieldRef()
        .withFieldPath(fieldPath)
        .endFieldRef()
        .endValueFrom()
        .build();
  }

  public static VolumeMount createVolumeMount(String name, String mountPath) {
    return createVolumeMount(name, mountPath, null);
  }

  public static VolumeMount createVolumeMount(String name, String mountPath, String subPath) {
    VolumeMountBuilder b = new VolumeMountBuilder()
        .withName(name)
        .withMountPath(mountPath);
    if (!isBlank(subPath)) {
      b.withSubPath(subPath);
    }
    return b.build();
  }

  public static Volume createVolumeFromHostPath(String name, String hostPath) {
    return new VolumeBuilder()
        .withName(name)
        .withHostPath(new HostPathVolumeSourceBuilder()
            .withPath(hostPath)
            .build())
        .build();
  }

  public static Volume createVolumeFromConfigMap(String name, String configMapName) {
    return new VolumeBuilder()
        .withName(name)
        .withConfigMap(new ConfigMapVolumeSourceBuilder()
            .withName(configMapName)
            .build())
        .build();
  }

  public static Quantity createQuantity(Number value, String format) {
    return new QuantityBuilder()
        .withAmount(String.valueOf(value))
        .withFormat(format)
        .build();
  }

  public static PolicyRule createPolicyRule(List<String> apiGroups,
                                            List<String> resources,
                                            List<String> verbs) {
    return new PolicyRuleBuilder()
        .withApiGroups(apiGroups)
        .withResources(resources)
        .withVerbs(verbs)
        .build();
  }

  public static PolicyRule createNonResourceURLsPolicyRule(List<String> urls,
                                                           List<String> verbs) {
    return new PolicyRuleBuilder()
        .withNonResourceURLs(urls)
        .withVerbs(verbs)
        .build();
  }

}
