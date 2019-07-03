package com.instana.operator.service;

import com.instana.operator.cache.CacheService;
import io.quarkus.runtime.StartupEvent;
import io.reactivex.plugins.RxJavaPlugins;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.annotation.Priority;
import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.event.Observes;

import static javax.interceptor.Interceptor.Priority.APPLICATION;

@ApplicationScoped
public class FatalErrorHandler {

  private static final Logger LOGGER = LoggerFactory.getLogger(CacheService.class);

  // TODO: If we use APPLICATION priority here, we must make sure that all other StartupEvent observers
  // have at minimum priority APPLICATION+1.
  public void onStartup(@Observes @Priority(APPLICATION) StartupEvent _ev) {
    RxJavaPlugins.setErrorHandler(exception -> {
      LOGGER.error(exception.getMessage(), exception);
      systemExit(-1);
    });
  }

  public void systemExit(int status) {
    System.exit(status);
  }
}
