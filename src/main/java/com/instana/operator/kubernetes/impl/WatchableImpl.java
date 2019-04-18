package com.instana.operator.kubernetes.impl;

import com.instana.operator.kubernetes.Closeable;
import com.instana.operator.kubernetes.Watchable;
import io.fabric8.kubernetes.api.model.HasMetadata;
import io.fabric8.kubernetes.api.model.KubernetesResourceList;
import io.fabric8.kubernetes.client.Watch;
import io.fabric8.kubernetes.client.Watcher;
import io.fabric8.kubernetes.client.dsl.WatchListDeletable;

public class WatchableImpl<T extends HasMetadata> implements Watchable<T> {

  private final WatchListDeletable<T, KubernetesResourceList<T>, Boolean, Watch, Watcher<T>> op;

  WatchableImpl(WatchListDeletable<T, KubernetesResourceList<T>, Boolean, Watch, Watcher<T>> op) {
   this.op = op;
  }

  @Override
  public Closeable watch(Watcher<T> watcher) {
    Watch w = op.watch(watcher);
    return w::close;
  }

  @Override
  public KubernetesResourceList<T> list() {
    return op.list();
  }
}
