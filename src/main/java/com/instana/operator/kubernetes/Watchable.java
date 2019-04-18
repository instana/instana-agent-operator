package com.instana.operator.kubernetes;

import io.fabric8.kubernetes.api.model.HasMetadata;
import io.fabric8.kubernetes.api.model.KubernetesResourceList;
import io.fabric8.kubernetes.client.Watcher;

public interface Watchable<T extends HasMetadata> {

  Closeable watch(Watcher<T> watcher);

  KubernetesResourceList<T> list();
}
