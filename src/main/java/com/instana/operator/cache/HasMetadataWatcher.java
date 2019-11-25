package com.instana.operator.cache;

import static com.instana.operator.cache.ExceptionHandlerWrapper.exitOnError;

import java.util.concurrent.ExecutorService;
import java.util.concurrent.RejectedExecutionException;
import java.util.function.BiConsumer;
import java.util.function.Consumer;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import com.instana.operator.FatalErrorHandler;

import io.fabric8.kubernetes.api.model.HasMetadata;
import io.fabric8.kubernetes.client.KubernetesClientException;
import io.fabric8.kubernetes.client.Watcher;

public class HasMetadataWatcher<T extends HasMetadata> implements Watcher<T> {
  private static final Logger LOGGER = LoggerFactory.getLogger(HasMetadataWatcher.class);

  private final ResourceMap<T> map;
  private final FatalErrorHandler fatalErrorHandler;
  private final ExecutorService executor;
  private final BiConsumer<Action, String> onEventCallback;
  private final Consumer<Exception> onErrorCallback;

  public HasMetadataWatcher(ResourceMap<T> map, FatalErrorHandler fatalErrorHandler, ExecutorService executor,
                            BiConsumer<Action, String> onEventCallback, Consumer<Exception> onErrorCallback) {
    this.map = map;
    this.fatalErrorHandler = fatalErrorHandler;
    this.executor = executor;
    this.onEventCallback = onEventCallback;
    this.onErrorCallback = onErrorCallback;
  }

  @Override
  public void eventReceived(Action action, T resource) {
    try {
      final String uid = resource.getMetadata().getUid();
      boolean updated = false;

      switch (action) {
      case ADDED:
        // fall through
      case MODIFIED:
        updated = map.putIfNewer(uid, resource);
        break;
      case DELETED:
        updated = map.remove(uid);
        break;
      default:
        LOGGER.error("Received unexpected " + action + " event for " + resource.getMetadata().getName());
        fatalErrorHandler.systemExit(-1);
      }

      if (updated) {
        executor.execute(() -> exitOnError(onEventCallback, fatalErrorHandler).accept(action, uid));
      }
    } catch (RejectedExecutionException e) {
      // This happens if executor.execute() throws a RejectedExecutionException, so it doesn't make sense to
      // schedule an onErrorCallback here. Just terminate the JVM and have Kubernetes restart the Pod.
      LOGGER.error(e.getMessage(), e);
      fatalErrorHandler.systemExit(-1);
    }
  }

  @Override
  public void onClose(KubernetesClientException cause) {
    if (cause != null) { // null means normal close, i.e. Watch.close() was called.
      try {
        // We call the onErrorCallback to allow for cleanup, but after that we terminate the JVM
        // and have Kubernetes restart the Pod, because there is no way to recover from this.
        executor.execute(() -> exitOnError(onErrorCallback, fatalErrorHandler)
            .andThen(c -> fatalErrorHandler.systemExit(-1))
            .accept(cause));
      } catch (Exception e) {
        // This happens if executor.execute() throws a RejectedExecutionException, so it doesn't make sense to
        // schedule an onErrorCallback here. Just terminate the JVM and have Kubernetes restart the Pod.
        LOGGER.error(e.getMessage(), e);
        fatalErrorHandler.systemExit(-1);
      }
    }
  }
}
