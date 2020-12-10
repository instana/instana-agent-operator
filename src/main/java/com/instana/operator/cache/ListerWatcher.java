package com.instana.operator.cache;

import io.fabric8.kubernetes.api.model.HasMetadata;
import io.fabric8.kubernetes.api.model.KubernetesResourceList;
import io.fabric8.kubernetes.client.Watch;
import io.fabric8.kubernetes.client.Watcher;
import io.fabric8.kubernetes.client.dsl.FilterWatchListDeletable;

/**
 * A ListerWatcher can list() and watch().
 */
public class ListerWatcher<T extends HasMetadata, L extends KubernetesResourceList<T>> {

  private final FilterWatchListDeletable<T, L, Boolean, Watch> op;

  ListerWatcher(FilterWatchListDeletable<T, L, Boolean, Watch> op) {
    this.op = op;
  }

  public KubernetesResourceList<T> list() {
    return op.list();
  }

  public Watch watch(Watcher<T> watcher) {
    return op.watch(watcher);
  }
}
