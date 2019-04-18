package com.instana.operator.kubernetes;

import io.fabric8.kubernetes.client.CustomResource;
import io.fabric8.kubernetes.client.CustomResourceList;

public interface CustomResourceClient<T extends CustomResource> {

  CustomResourceList<T> list();

  T createOrUpdate(T resource);
}
