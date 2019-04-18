package com.instana.operator.kubernetes.impl;

import com.instana.operator.customresource.DoneableElectedLeader;
import com.instana.operator.customresource.ElectedLeader;
import com.instana.operator.customresource.ElectedLeaderList;
import com.instana.operator.kubernetes.Client;
import com.instana.operator.kubernetes.CustomResourceClient;
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

public class ClientImpl implements Client {

  private final DefaultKubernetesClient client;

  public ClientImpl(DefaultKubernetesClient client) {
    this.client = client;
  }

  @Override
  @SuppressWarnings("unchecked")
  public <T extends HasMetadata> T create(String namespace, KubernetesResource<T> resource) {
    if (resource instanceof ServiceAccount) {
      return (T) client.serviceAccounts().inNamespace(namespace).create((ServiceAccount) resource);
    }
    if (resource instanceof ClusterRole) {
      return (T) client.rbac().clusterRoles().inNamespace(namespace).create((ClusterRole) resource);
    }
    if (resource instanceof ClusterRoleBinding) {
      return (T) client.rbac().clusterRoleBindings().inNamespace(namespace).create((ClusterRoleBinding) resource);
    }
    if (resource instanceof Secret) {
      return (T) client.secrets().inNamespace(namespace).create((Secret) resource);
    }
    if (resource instanceof DaemonSet) {
      return (T) client.apps().daemonSets().inNamespace(namespace).create((DaemonSet) resource);
    }
    if (resource instanceof ConfigMap) {
      return (T) client.configMaps().inNamespace(namespace).create((ConfigMap) resource);
    }
    if (resource instanceof Event) {
      return (T) client.events().inNamespace(namespace).create((Event) resource);
    }
    throw new IllegalArgumentException("Resource type " + resource.getClass().getSimpleName() + " not implemented yet.");
  }

  @Override
  @SuppressWarnings("unchecked")
  public <T extends HasMetadata> T get(String namespace, String name, Class<T> t) {
    if (ConfigMap.class.isAssignableFrom(t)) {
      return (T) client.configMaps().inNamespace(namespace).withName(name).get();
    }
    if (CustomResourceDefinition.class.isAssignableFrom(t)) {
      CustomResourceDefinitionList list = client.inNamespace(namespace).customResourceDefinitions().list();
      if (list != null) {
        return (T) list.getItems().stream().filter(crd -> name.equals(crd.getMetadata().getName())).findAny().orElse(null);
      } else {
        return null;
      }
    }
    if (ReplicaSet.class.isAssignableFrom(t)) {
      return (T) client.apps().replicaSets().inNamespace(namespace).withName(name).get();
    }
    throw new IllegalArgumentException("Resource type " + t.getSimpleName() + " not implemented yet.");
  }

  @Override
  public String getApiVersion() {
    return client.getApiVersion();
  }

  @Override
  @SuppressWarnings("unchecked")
  public <T extends CustomResource> CustomResourceClient<T> makeCustomResourceClient(String namespace,
                                                                                     CustomResourceDefinition crd,
                                                                                     Class<T> customResourceClass) {
    if (customResourceClass.isAssignableFrom(ElectedLeader.class)) {
      return new CustomResourceClientImpl(
          client
          .customResources(crd, ElectedLeader.class, ElectedLeaderList.class, DoneableElectedLeader.class)
          .inNamespace(namespace)
      );
    }
    throw new IllegalArgumentException("Resource type " + customResourceClass.getSimpleName() + " not implemented yet.");
  }

  @Override
  @SuppressWarnings("unchecked")
  public <T extends HasMetadata> Watchable<T> watch(String namespace, Class<T> resourceClass) {
    if (ConfigMap.class.isAssignableFrom(resourceClass)) {
      return new WatchableImpl(client.configMaps().inNamespace(namespace));
    }
    if (Pod.class.isAssignableFrom(resourceClass)) {
      return new WatchableImpl(client.pods().inNamespace(namespace));
    }
    throw new IllegalArgumentException("Resource type " + resourceClass.getSimpleName() + " not implemented yet.");
  }

  @Override
  @SuppressWarnings("unchecked")
  public <T extends HasMetadata> Watchable<T> watch(String namespace, Label label, Class<T> resourceClass) {
    if (Pod.class.isAssignableFrom(resourceClass)) {
      return new WatchableImpl(client.pods().inNamespace(namespace).withLabel(label.getName(), label.getValue()));
    }
    throw new IllegalArgumentException("Resource type " + resourceClass.getSimpleName() + " not implemented yet.");
  }

  @Override
  public void close() {
    client.close();
  }
}
