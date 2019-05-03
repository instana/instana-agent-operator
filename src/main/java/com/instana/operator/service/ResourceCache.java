package com.instana.operator.service;

import com.instana.operator.GlobalErrorEvent;
import com.instana.operator.kubernetes.Closeable;
import com.instana.operator.kubernetes.Watchable;
import io.fabric8.kubernetes.api.model.HasMetadata;
import io.fabric8.kubernetes.api.model.KubernetesResourceList;
import io.fabric8.kubernetes.client.KubernetesClientException;
import io.fabric8.kubernetes.client.Watcher;
import io.reactivex.Observable;
import io.reactivex.ObservableSource;
import io.reactivex.Observer;
import io.reactivex.disposables.CompositeDisposable;
import io.reactivex.disposables.Disposable;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.enterprise.event.Event;
import java.util.*;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicReference;
import java.util.stream.Collectors;

import static java.util.Collections.emptyList;

public class ResourceCache<T extends HasMetadata> {

  private final ConcurrentHashMap<String, T> entries = new ConcurrentHashMap<>(16, 0.9f, 1);

  private final Logger logger;
  private final Watchable watchList;
  private final Event<GlobalErrorEvent> errorEvent;

  public ResourceCache(String name, Watchable watchable, Event<GlobalErrorEvent> errorEvent) {
    this.logger = LoggerFactory.getLogger(ResourceCache.class.getName() + "." + name);
    this.watchList = watchable;
    this.errorEvent = errorEvent;
  }

  public Optional<T> get(String name) {
    return Optional.ofNullable(entries.get(name));
  }

  public Observable<ChangeEvent<T>> observe() {
    return Observable.defer(() -> new ObservableSource<ChangeEvent<T>>() {
      @Override
      public void subscribe(Observer<? super ChangeEvent<T>> observer) {
        AtomicReference<Closeable> watch = new AtomicReference<>();
        Disposable watchInterval = Observable.interval(0, 1, TimeUnit.MINUTES)
            .doOnDispose(() -> {
              Closeable w = watch.getAndSet(null);
              if (null != w) {
                logger.debug("Closing old watch");
                w.close();
              }
            })
            .subscribe(now -> {
              Closeable w = watch.get();
              if (null != w) {
                logger.debug("Closing previous watch");
                w.close();
              }

              watch.set(watchList.watch(new Watcher<T>() {
                @Override
                public void eventReceived(Action action, T resource) {
                  ChangeEvent<T> change = null;
                  switch (action) {
                  case ADDED:
                    change = new ChangeEvent<>(resource.getMetadata().getName(), null, resource);
                  case MODIFIED:
                    T oldVal = entries.put(resource.getMetadata().getName(), resource);
                    if (null != oldVal) {
                      String oldValVers = oldVal.getMetadata().getResourceVersion();
                      String newValVers = resource.getMetadata().getResourceVersion();
                      if (!Objects.equals(oldValVers, newValVers)) {
                        change = new ChangeEvent<>(resource.getMetadata().getName(), oldVal, resource);
                      }
                    }
                    break;
                  case DELETED:
                    if (null != (oldVal = entries.remove(resource.getMetadata().getName()))) {
                      change = new ChangeEvent<>(resource.getMetadata().getName(), oldVal, null);
                    }
                    break;
                  case ERROR:
                    errorEvent.fire(new GlobalErrorEvent("Resource " + resource.getMetadata().getName() + " encountered an error."));
                    break;
                  }

                  if (null != change) {
                    observer.onNext(change);
                  }
                }

                @Override
                public void onClose(KubernetesClientException cause) {
                  if (null != cause) {
                    logger.debug("Handling error: {}", cause.getMessage(), cause);
                    observer.onError(cause.getCause());
                  }
                  // don't close the observer.
                }
              }));
            });

        Disposable listInterval = Observable.interval(0, 1, TimeUnit.MINUTES)
            .subscribe(now -> {
              KubernetesResourceList<T> krl;
              try {
                krl = watchList.list();
              } catch (KubernetesClientException e) {
                errorEvent.fire(new GlobalErrorEvent(e));
                return;
              }

              List<T> resourceList;
              if (null == krl) {
                resourceList = emptyList();
              } else {
                resourceList = krl.getItems();
              }
              logger.debug("Found {}[{}]. Now reconciling the cache...",
                  krl.getClass().getSimpleName(),
                  resourceList.size());

              Map<String, T> incomingResources = resourceList.stream()
                  .filter(r -> r.getMetadata() != null)
                  .filter(r -> r.getMetadata().getName() != null)
                  .collect(Collectors.toMap(r -> r.getMetadata().getName(), r -> r));

              synchronized (entries) {
                entries.keySet().removeIf(name -> {
                  if (incomingResources.containsKey(name)) {
                    return false;
                  }
                  observer.onNext(new ChangeEvent<>(name, entries.get(name), null));
                  return true;
                });

                incomingResources.forEach((name, newVal) -> {
                  T oldVal = entries.put(name, newVal);
                  ChangeEvent<T> changeEvent = null;
                  if (null == oldVal
                      || !oldVal.getMetadata().getResourceVersion().equals(newVal.getMetadata().getResourceVersion())) {
                    logger.debug("Publishing {} {} event for {}",
                        newVal.getKind(),
                        (null == oldVal ? "added" : "modified"),
                        name);
                    changeEvent = new ChangeEvent<>(name, oldVal, newVal);
                  }
                  if (null != changeEvent) {
                    observer.onNext(changeEvent);
                  }
                });
              }
            });

        observer.onSubscribe(new CompositeDisposable(watchInterval, listInterval));
      }
    });
  }

  public List<T> toList() {
    synchronized (entries) {
      return new ArrayList<>(entries.values());
    }
  }

  public class ChangeEvent<T> {
    private final String name;
    private final T previousValue;
    private final T nextValue;

    private ChangeEvent(String name, T previousValue, T nextValue) {
      this.name = name;
      this.previousValue = previousValue;
      this.nextValue = nextValue;
    }

    public String getName() {
      return name;
    }

    public T getPreviousValue() {
      return previousValue;
    }

    public T getNextValue() {
      return nextValue;
    }

    public boolean isAdded() {
      return previousValue == null && nextValue != null;
    }

    public boolean isModified() {
      return previousValue != null && nextValue != null;
    }

    public boolean isDeleted() {
      return previousValue != null && nextValue == null;
    }

    @Override
    public String toString() {
      return "ChangeEvent{" +
          "name='" + name + '\'' +
          ", previousValue=" + previousValue +
          ", nextValue=" + nextValue +
          '}';
    }
  }

}
