package com.instana.operator.service;

import static java.util.Collections.emptyList;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Optional;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.TimeUnit;
import java.util.function.Consumer;
import java.util.stream.Collectors;

import javax.enterprise.event.Event;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import com.instana.operator.GlobalErrorEvent;

import io.fabric8.kubernetes.api.model.HasMetadata;
import io.fabric8.kubernetes.api.model.KubernetesResourceList;
import io.fabric8.kubernetes.client.KubernetesClientException;
import io.fabric8.kubernetes.client.Watch;
import io.fabric8.kubernetes.client.Watcher;
import io.fabric8.kubernetes.client.dsl.WatchListDeletable;
import io.reactivex.Observable;
import io.reactivex.ObservableSource;
import io.reactivex.Observer;
import io.reactivex.disposables.Disposable;

public class ResourceCache<T extends HasMetadata> implements Disposable {

  private final ConcurrentHashMap<String, T> entries = new ConcurrentHashMap<>(16, 0.9f, 1);

  private final Logger logger;
  private final WatchListDeletable<T, ? extends KubernetesResourceList<T>, ?, Watch, Watcher<T>> watchList;
  private final Event<GlobalErrorEvent> errorEvent;
  private final Consumer<ResourceCache<T>> onDispose;

  private Disposable interval;

  private volatile Watch watch;

  public ResourceCache(String name,
                       WatchListDeletable<T, ? extends KubernetesResourceList<T>, ?, Watch, Watcher<T>> watchList,
                       Event<GlobalErrorEvent> errorEvent,
                       Consumer<ResourceCache<T>> onDispose) {
    this.logger = LoggerFactory.getLogger(ResourceCache.class.getName() + "." + name);
    this.watchList = watchList;
    this.errorEvent = errorEvent;
    this.onDispose = onDispose;
  }

  @Override
  public synchronized void dispose() {
    if (null != interval) {
      interval.dispose();
      interval = null;
    }
    if (null != watch) {
      watch.close();
      watch = null;
    }
    onDispose.accept(this);
  }

  @Override
  public boolean isDisposed() {
    return null != interval && null != watch;
  }

  public Optional<T> get(String name) {
    return Optional.ofNullable(entries.get(name));
  }

  public Observable<ChangeEvent<T>> observe() {
    return Observable.defer(() -> new ObservableSource<ChangeEvent<T>>() {
      @Override
      public void subscribe(Observer<? super ChangeEvent<T>> observer) {
        logger.debug("subscribe()");
        interval = Observable.interval(0, 1, TimeUnit.MINUTES)
            .subscribe(now -> {
              KubernetesResourceList<T> krl;
              try {
                krl = watchList.list();
              } catch (KubernetesClientException e) {
                errorEvent.fire(new GlobalErrorEvent(e.getCause()));
                return;
              }

              List<T> resourceList;
              if (null == krl) {
                resourceList = emptyList();
              } else {
                resourceList = krl.getItems();
              }
              logger.debug("Found {} {}. Now reconciling the cache...",
                  resourceList.size(),
                  krl.getClass().getSimpleName());

              Map<String, T> incomingResources = resourceList.stream()
                  .filter(r -> r.getMetadata() != null)
                  .filter(r -> r.getMetadata().getName() != null)
                  .collect(Collectors.toMap(r -> r.getMetadata().getName(), r -> r));

              synchronized (entries) {
                incomingResources.forEach((name, newVal) -> {
                  T oldVal = entries.put(name, newVal);
                  ChangeEvent<T> changeEvent = null;
                  if (null == oldVal
                      || !oldVal.getMetadata().getResourceVersion().equals(newVal.getMetadata().getResourceVersion())) {
                    logger.debug("Publishing {} {} ChangeEvent {}",
                        (null == oldVal ? "added" : "modified"),
                        newVal.getKind(),
                        name);
                    changeEvent = new ChangeEvent<>(name, oldVal, newVal);
                  }
                  if (null != changeEvent) {
                    observer.onNext(changeEvent);
                  }
                });

                entries.keySet().removeIf(name -> {
                  if (incomingResources.containsKey(name)) {
                    return false;
                  }
                  observer.onNext(new ChangeEvent<>(name, entries.get(name), null));
                  return true;
                });
              }

              if (null != watch) {
                return;
              }
              watch = watchList.watch(new Watcher<T>() {
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
                    errorEvent.fire(new GlobalErrorEvent(new IllegalStateException(
                        "Resource " + resource.getMetadata().getName() + " encountered an error.")));
                    break;
                  }

                  if (null != change) {
                    observer.onNext(change);
                  }
                }

                @Override
                public void onClose(KubernetesClientException cause) {
                  if (null != cause) {
                    logger.debug("Handling error: {}", cause.getCause());
                    observer.onError(cause.getCause());
                  }
                  observer.onComplete();
                }
              });
            });

        observer.onSubscribe(ResourceCache.this);
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
  }

}
