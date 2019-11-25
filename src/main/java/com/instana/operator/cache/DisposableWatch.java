package com.instana.operator.cache;

import java.util.concurrent.atomic.AtomicReference;

import io.fabric8.kubernetes.client.Watch;
import io.reactivex.disposables.Disposable;

class DisposableWatch implements Disposable {

  private final AtomicReference<Watch> watch = new AtomicReference<>();

  DisposableWatch(Watch watch) {
    this.watch.set(watch);
  }

  @Override
  public void dispose() {
    Watch w = watch.getAndSet(null);
    if (w != null) {
      w.close();
    }
  }

  @Override
  public boolean isDisposed() {
    return watch.get() == null;
  }
}
