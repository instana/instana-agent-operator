package com.instana.operator.kubernetes.impl;

import com.instana.operator.kubernetes.CustomResourceClient;
import io.fabric8.kubernetes.client.CustomResource;
import io.fabric8.kubernetes.client.CustomResourceDoneable;
import io.fabric8.kubernetes.client.CustomResourceList;
import io.fabric8.kubernetes.client.dsl.NonNamespaceOperation;
import io.fabric8.kubernetes.client.dsl.Resource;

public class CustomResourceClientImpl<T extends CustomResource> implements CustomResourceClient<T> {

  private final NonNamespaceOperation<T, ? extends CustomResourceList<T>, ? extends CustomResourceDoneable<T>, ? extends Resource<T, ? extends CustomResourceDoneable<T>>> op;

  CustomResourceClientImpl(NonNamespaceOperation<T, ? extends CustomResourceList<T>, ? extends CustomResourceDoneable<T>, ? extends Resource<T, ? extends CustomResourceDoneable<T>>> op) {
    this.op = op;
  }

  @Override
  public CustomResourceList<T> list() {
    return op.list();
  }

  @Override
  @SuppressWarnings("unchecked")
  public T createOrUpdate(T resource) {
    return op.createOrReplace(resource);
  }
}
