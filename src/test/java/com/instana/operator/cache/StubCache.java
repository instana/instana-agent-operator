/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */
package com.instana.operator.cache;

import io.fabric8.kubernetes.api.model.HasMetadata;
import io.fabric8.kubernetes.api.model.KubernetesResourceList;
import io.fabric8.kubernetes.client.Watch;
import io.fabric8.kubernetes.client.dsl.FilterWatchListDeletable;
import io.reactivex.Observable;
import io.reactivex.Observer;

public class StubCache<T extends HasMetadata, L extends KubernetesResourceList<T>> extends Cache<T, L> {

  public StubCache() {
    super(null, null);
  }

  @Override
  public Observable<CacheEvent> listThenWatch(FilterWatchListDeletable<T, L, Boolean, Watch> op) {
    return new Observable<CacheEvent>() {
      @Override
      protected void subscribeActual(Observer<? super CacheEvent> observer) {
      }

    };
  }

  @Override
  public Observable<CacheEvent> listThenWatch(ListerWatcher<T, L> op) {
    return new Observable<CacheEvent>() {
      @Override
      protected void subscribeActual(Observer<? super CacheEvent> observer) {

      }
    };
  }
}
