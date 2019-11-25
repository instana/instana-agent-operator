package com.instana.operator.cache;

import com.instana.operator.FatalErrorHandler;
import io.fabric8.kubernetes.api.model.HasMetadata;
import io.fabric8.kubernetes.api.model.KubernetesResourceList;
import io.fabric8.kubernetes.client.Watch;
import io.fabric8.kubernetes.client.Watcher;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.concurrent.ExecutorService;
import java.util.function.BiConsumer;
import java.util.function.Consumer;

import static com.instana.operator.cache.ExceptionHandlerWrapper.exitOnError;

class ListThenWatchOperation {

  private static final Logger LOGGER = LoggerFactory.getLogger(ListThenWatchOperation.class);

  /**
   * Note that the callback is called in the executor thread.
   */
  static <T extends HasMetadata, L extends KubernetesResourceList<T>> Watch run(ExecutorService executor,
                                                                                ResourceMap<T> map,
                                                                                ListerWatcher<T, L> op,
                                                                                FatalErrorHandler fatalErrorHandler,
                                                                                BiConsumer<Watcher.Action, String> onEventCallback,
                                                                                Consumer<Exception> onErrorCallback) {

    // list

    op.list()
        .getItems()
        .forEach(resource -> {
              map.putIfNewer(resource.getMetadata().getUid(), resource);
              String uid = resource.getMetadata().getUid();
              executor.execute(() -> exitOnError(onEventCallback, fatalErrorHandler).accept(Watcher.Action.ADDED, uid));
            }
        );

    // watch

    return op.watch(new HasMetadataWatcher<>(map, fatalErrorHandler, executor, onEventCallback, onErrorCallback));
  }

}
