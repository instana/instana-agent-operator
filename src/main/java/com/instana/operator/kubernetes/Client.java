package com.instana.operator.kubernetes;

import io.fabric8.kubernetes.api.model.HasMetadata;
import io.fabric8.kubernetes.api.model.KubernetesResource;
import io.fabric8.kubernetes.api.model.KubernetesResourceList;
import io.fabric8.kubernetes.api.model.apiextensions.CustomResourceDefinition;
import io.fabric8.kubernetes.client.CustomResource;

public interface Client extends Closeable {

  String getApiVersion();

  <T extends CustomResource> void registerCustomResource(String namespace, CustomResourceDefinition crd, Class<T> customResourceClass);

  <T extends HasMetadata> T create(String namespace, KubernetesResource<T> resource);

  <T extends HasMetadata> T get(String namespace, String name, Class<T> t);

  <T extends HasMetadata> Watchable<T> watch(String namespace, Class<T> resourceClass);

  <T extends HasMetadata> Watchable<T> watch(String namespace, Label label, Class<T> resourceClass);

  <T extends HasMetadata> KubernetesResourceList<T> list(Class<T> resourceClass);

  <T extends HasMetadata> T createOrUpdate(T resource);
}
