package com.instana.operator.cache;

import com.instana.operator.FatalErrorHandler;
import io.fabric8.kubernetes.api.model.HasMetadata;
import io.fabric8.kubernetes.api.model.KubernetesResourceList;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.enterprise.context.ApplicationScoped;
import javax.inject.Inject;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;

@ApplicationScoped
public class CacheService {

  private static final Logger LOGGER = LoggerFactory.getLogger(CacheService.class);

  @Inject
  FatalErrorHandler fatalErrorHandler;

  // All events will be scheduled to this single thread. Subscribers don't need to be thread save.
  // Note on the uncaught exception handler: This only helps if exceptions are not caught :)
  // If you call executor.submit(), exceptions will be caught and wrapped into the Future,
  // so the uncaught exception handler will not be called.
  private final ExecutorService executor = Executors.newSingleThreadExecutor(runnable -> {
    Thread thread = Executors.defaultThreadFactory().newThread(runnable);
    thread.setDaemon(true);
    thread.setName("k8s-handler");
    // For a warning on the uncaught Exception handler, see comment on ExecutorProducer.
    thread.setUncaughtExceptionHandler((t, e) -> {
      LOGGER.error(e.getMessage(), e);
      fatalErrorHandler.systemExit(-1);
    });
    return thread;
  });

  public <T extends HasMetadata, L extends KubernetesResourceList<T>> Cache<T, L> newCache(Class<T> resourceClass, Class<L> resourceListClass) {
    return new Cache<>(executor, fatalErrorHandler);
  }

  // For tests only. In production the CacheService is a Singleton that should never be terminated.
  void terminate(long timeout, TimeUnit unit) throws Exception {
    executor.shutdown();
    if (!executor.awaitTermination(timeout, unit)) {
      throw new Exception("timeout while terminating the cache service executor thread");
    }
  }
}
