package com.instana.operator.kubernetes.impl;

import com.instana.operator.customresource.*;
import com.instana.operator.kubernetes.Client;
import com.instana.operator.kubernetes.Label;
import com.instana.operator.kubernetes.Watchable;
import io.fabric8.kubernetes.api.model.*;
import io.fabric8.kubernetes.api.model.apiextensions.CustomResourceDefinition;
import io.fabric8.kubernetes.api.model.apiextensions.CustomResourceDefinitionList;
import io.fabric8.kubernetes.api.model.apps.DaemonSet;
import io.fabric8.kubernetes.api.model.apps.ReplicaSet;
import io.fabric8.kubernetes.api.model.rbac.ClusterRole;
import io.fabric8.kubernetes.api.model.rbac.ClusterRoleBinding;
import io.fabric8.kubernetes.client.CustomResource;
import io.fabric8.kubernetes.client.DefaultKubernetesClient;
import io.fabric8.kubernetes.client.dsl.MixedOperation;

import java.util.Collections;
import java.util.HashMap;
import java.util.Map;

public class ClientImpl implements Client {

  private final DefaultKubernetesClient defaultClient;
  private final Map<Class<? extends CustomResource>, MixedOperation> customResourceClients;

  public ClientImpl(DefaultKubernetesClient client) {
    this.defaultClient = client;
    this.customResourceClients = Collections.synchronizedMap(new HashMap<>());
  }

  @Override
  @SuppressWarnings("unchecked")
  public <T extends HasMetadata> T create(String namespace, KubernetesResource<T> resource) {
    if (resource instanceof ServiceAccount) {
      return (T) defaultClient.serviceAccounts().inNamespace(namespace).create((ServiceAccount) resource);
    }
    if (resource instanceof ClusterRole) {
      return (T) defaultClient.rbac().clusterRoles().inNamespace(namespace).create((ClusterRole) resource);
    }
    if (resource instanceof ClusterRoleBinding) {
      return (T) defaultClient.rbac().clusterRoleBindings().inNamespace(namespace).create((ClusterRoleBinding) resource);
    }
    if (resource instanceof Secret) {
      return (T) defaultClient.secrets().inNamespace(namespace).create((Secret) resource);
    }
    if (resource instanceof DaemonSet) {
      return (T) defaultClient.apps().daemonSets().inNamespace(namespace).create((DaemonSet) resource);
    }
    if (resource instanceof ConfigMap) {
      return (T) defaultClient.configMaps().inNamespace(namespace).create((ConfigMap) resource);
    }
    if (resource instanceof Event) {
      return (T) defaultClient.events().inNamespace(namespace).create((Event) resource);
    }
    throw new IllegalArgumentException("Resource type " + resource.getClass().getSimpleName() + " not implemented yet.");
  }

  @Override
  @SuppressWarnings("unchecked")
  public <T extends HasMetadata> T get(String namespace, String name, Class<T> t) {
    if (ConfigMap.class.isAssignableFrom(t)) {
      return (T) defaultClient.configMaps().inNamespace(namespace).withName(name).get();
    }
    if (CustomResourceDefinition.class.isAssignableFrom(t)) {
      CustomResourceDefinitionList list = defaultClient.inNamespace(namespace).customResourceDefinitions().list();
      if (list != null) {
        return (T) list.getItems().stream().filter(crd -> name.equals(crd.getMetadata().getName())).findAny().orElse(null);
      } else {
        return null;
      }
    }
    if (ReplicaSet.class.isAssignableFrom(t)) {
      return (T) defaultClient.apps().replicaSets().inNamespace(namespace).withName(name).get();
    }
    throw new IllegalArgumentException("Resource type " + t.getSimpleName() + " not implemented yet.");
  }

  @Override
  public <T extends HasMetadata>  KubernetesResourceList<T> list(Class<T> resourceClass) {
    MixedOperation client = customResourceClients.get(resourceClass);
    if (client == null) {
      throw new IllegalArgumentException("Resource type " + resourceClass.getSimpleName() + " not implemented yet.");
    }
    return (KubernetesResourceList<T>) client.list();
  }

  @Override
  @SuppressWarnings("unchecked")
  public <T extends HasMetadata> T createOrUpdate(T resource) {
    MixedOperation client = customResourceClients.get(resource.getClass());
    if (client == null) {
      throw new IllegalArgumentException("Resource type " + resource.getClass().getSimpleName() + " not implemented yet.");
    }
    return (T) client.createOrReplace(resource);
  }

  @Override
  public String getApiVersion() {
    return defaultClient.getApiVersion();
  }

  @Override
  public <T extends CustomResource> void registerCustomResource(String namespace, CustomResourceDefinition crd,
                                                                Class<T> customResourceClass) {
    if (customResourceClass.isAssignableFrom(InstanaAgent.class)) {
      customResourceClients.put(InstanaAgent.class, defaultClient.customResources(crd, InstanaAgent.class, InstanaAgentList.class, DoneableInstanaAgent.class));
    } else {
      throw new IllegalArgumentException("Resource type " + customResourceClass.getSimpleName() + " not implemented yet.");
    }
  }

  @Override
  @SuppressWarnings("unchecked")
  public <T extends HasMetadata> Watchable<T> watch(String namespace, Class<T> resourceClass) {
    if (ConfigMap.class.isAssignableFrom(resourceClass)) {
      return new WatchableImpl(defaultClient.configMaps().inNamespace(namespace));
    }
    if (Pod.class.isAssignableFrom(resourceClass)) {
      return new WatchableImpl(defaultClient.pods().inNamespace(namespace));
    }
    MixedOperation client = customResourceClients.get(resourceClass);
    if (client != null) {
      return new WatchableImpl<>(client);
    }
    throw new IllegalArgumentException("Resource type " + resourceClass.getSimpleName() + " not implemented yet.");
  }

  @Override
  @SuppressWarnings("unchecked")
  public <T extends HasMetadata> Watchable<T> watch(String namespace, Label label, Class<T> resourceClass) {
    if (Pod.class.isAssignableFrom(resourceClass)) {
      return new WatchableImpl(defaultClient.pods().inNamespace(namespace).withLabel(label.getName(), label.getValue()));
    }
    throw new IllegalArgumentException("Resource type " + resourceClass.getSimpleName() + " not implemented yet.");
  }

  @Override
  public void close() {
    customResourceClients.clear();
    defaultClient.close();
  }
}
