/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc.
 */
package com.instana.operator;

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

  public <T> T logAndExit(Throwable t) {
    LOGGER.error("Uncaught exception in CDI event handler: " + t.getMessage(), t);
    systemExit(-1);
    return null; // will not happen, because we called System.exit();
  }

  // This must happen before other onStartup() methoods, therefore we set priority APPLICATION - 2.
  public void onStartup(@Observes @Priority(APPLICATION - 2) StartupEvent _ev) {
    RxJavaPlugins.setErrorHandler(exception -> {
      LOGGER.error(exception.getMessage(), exception);
      systemExit(-1);
    });
  }

  public void systemExit(int status) {
    System.exit(status);
  }
}
