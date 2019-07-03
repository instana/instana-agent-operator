package com.instana.operator.cache;

import com.instana.operator.service.FatalErrorHandler;
import io.fabric8.kubernetes.api.model.HasMetadata;
import io.fabric8.kubernetes.api.model.KubernetesResourceList;
import io.fabric8.kubernetes.client.Watch;
import io.fabric8.kubernetes.client.Watcher;
import io.fabric8.kubernetes.client.dsl.FilterWatchListDeletable;
import io.reactivex.Observable;
import io.reactivex.Observer;

import java.util.Optional;
import java.util.concurrent.ExecutorService;
import java.util.function.BiConsumer;
import java.util.function.Consumer;

import static com.instana.operator.cache.ExceptionHandlerWrapper.exitOnError;

public class Cache<T extends HasMetadata, L extends KubernetesResourceList<T>> {

  private final ResourceMap<T> map = new ResourceMap<>();
  private final ExecutorService executor;
  private final FatalErrorHandler fatalErrorHandler;

  /**
   * package private, use {@link CacheService} to create a new Cache.
   */
  Cache(ExecutorService executor, FatalErrorHandler fatalErrorHandler) {
    this.executor = executor;
    this.fatalErrorHandler = fatalErrorHandler;
  }

  public Optional<T> get(String uid) {
    return map.get(uid);
  }

  public Observable<CacheEvent> listThenWatch(FilterWatchListDeletable<T, L, Boolean, Watch, Watcher<T>> op) {
    return listThenWatch(new ListerWatcher<>(op));
  }

  public Observable<CacheEvent> listThenWatch(ListerWatcher<T, L> op) {
    return new Observable<CacheEvent>() {
      @Override
      protected void subscribeActual(Observer<? super CacheEvent> observer) {
        BiConsumer<Watcher.Action, String> onEventCallback = (action, uid) -> observer.onNext(new CacheEvent(action, uid));
        Consumer<Exception> onErrorCallback = observer::onError;
        try {
          ListThenWatchOperation.run(executor, map, op, fatalErrorHandler, onEventCallback, onErrorCallback);
        } catch (Exception e) {
          // First call the error callback to provide a hook for cleanup, but then call System.exit(-1) because
          // we won't continue from here. Let Kubernetes restart the Pod.
          exitOnError(onErrorCallback, fatalErrorHandler).andThen(ex -> fatalErrorHandler.systemExit(-1)).accept(e);
        }
      }
    };
  }
}
