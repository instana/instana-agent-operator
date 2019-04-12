package com.instana.operator.service;

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

import com.instana.operator.GlobalErrorEvent;

import io.fabric8.kubernetes.api.model.HasMetadata;
import io.fabric8.kubernetes.api.model.KubernetesResourceList;
import io.fabric8.kubernetes.client.KubernetesClientException;
import io.fabric8.kubernetes.client.Watch;
import io.fabric8.kubernetes.client.Watcher;
import io.fabric8.kubernetes.client.dsl.WatchListDeletable;
import io.reactivex.Observable;
import io.reactivex.disposables.Disposable;
import io.reactivex.schedulers.Schedulers;
import io.reactivex.subjects.PublishSubject;

public class ResourceCache<T extends HasMetadata> implements Disposable {

  private final Map<String, T> entries = new ConcurrentHashMap<>();
  private final PublishSubject<ChangeEvent<T>> publisher = PublishSubject.create();

  private final Consumer<ResourceCache<T>> onDispose;

  private Disposable interval;

  private volatile Watch watch;

  public ResourceCache(WatchListDeletable<T, ? extends KubernetesResourceList<T>, ?, Watch, Watcher<T>> watchList,
                       Event<GlobalErrorEvent> errorEvent,
                       Consumer<ResourceCache<T>> onDispose) {
    this.onDispose = onDispose;

    this.interval = Observable.interval(0, 5, TimeUnit.MINUTES)
        .subscribe(now -> {
          if (null != watch) {
            synchronized (ResourceCache.this) {
              if (null != watch) {
                watch.close();
                watch = null;
              }
            }
          }

          KubernetesResourceList<T> resourceList;
          try {
            resourceList = watchList.list();
          } catch (KubernetesClientException e) {
            errorEvent.fire(new GlobalErrorEvent(e.getCause()));
            return;
          }

          if (null == resourceList || resourceList.getItems().isEmpty()) {
            entries.clear();
          }

          Map<String, T> newResources = resourceList.getItems().stream()
              .collect(Collectors.toMap(r -> r.getMetadata().getName(), r -> r));

          newResources.forEach((name, newVal) -> {
            T oldVal = entries.put(name, newVal);
            if (oldVal != newVal) {
              publisher.onNext(new ChangeEvent<>(name, oldVal, newVal));
            }
          });

          entries.keySet().removeIf(name -> {
            if (newResources.containsKey(name)) {
              return false;
            }
            publisher.onNext(new ChangeEvent<>(name, entries.remove(name), null));
            return true;
          });

          List<T> resources = new ArrayList<>(entries.values());
          String resourceVersion = resources.size() > 0
              ? resources.get(resources.size() - 1).getMetadata().getResourceVersion()
              : null;

          watch = watchList.withResourceVersion(resourceVersion).watch(new Watcher<T>() {
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
                  change = new ChangeEvent<>(resource.getMetadata().getName(), resource, null);
                }
                break;
              case ERROR:
                errorEvent.fire(new GlobalErrorEvent(new IllegalStateException(
                    "Resource " + resource.getMetadata().getName() + " encountered an error.")));
                break;
              }

              if (null != change) {
                publisher.onNext(change);
              }
            }

            @Override
            public void onClose(KubernetesClientException cause) {
              if (null != cause) {
                publisher.onError(cause.getCause());
              }
              publisher.onComplete();
            }
          });
        });
  }

  @Override
  public void dispose() {
    if (null != interval) {
      interval.dispose();
    }
    if (null != watch) {
      watch.close();
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
    return publisher.observeOn(Schedulers.io());
  }

  public List<T> toList() {
    return new ArrayList<>(entries.values());
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
  }

}
