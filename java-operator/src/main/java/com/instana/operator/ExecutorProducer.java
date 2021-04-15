/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc.
 */
package com.instana.operator;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.enterprise.event.NotificationOptions;
import javax.enterprise.inject.Produces;
import javax.inject.Named;
import javax.inject.Singleton;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.ThreadFactory;

import static java.util.concurrent.Executors.newSingleThreadScheduledExecutor;

public class ExecutorProducer {

  private static final Logger LOGGER = LoggerFactory.getLogger(ExecutorProducer.class);
  public static final String CDI_HANDLER = "cdi-handler";
  public static final String AGENT_COORDINATOR_POLL = "agent-coordinator-poll";

  /**
   * The CDI_HANDLER is a single-threaded executor. It should be used to handle CDI events.
   * That way, CDI events are handled sequentially in a single thread, and business logic
   * does not need to be thread save.
   * This is similar to Weld's SERIAL notification mode.
   * See https://docs.jboss.org/weld/reference/latest/en-US/html/events.html
   */
  @Produces
  @Singleton
  public NotificationOptions cdiHandler(@Named(CDI_HANDLER) ScheduledExecutorService executor) {
    return NotificationOptions.ofExecutor(executor);
  }

  @Produces
  @Singleton
  @Named(CDI_HANDLER)
  public ScheduledExecutorService cdiHandler(SingleThreadFactoryBuilder threadFactoryBuilder) {
    return newSingleThreadScheduledExecutor(threadFactoryBuilder.build("cdi-handler"));
  }

  @Produces
  @Singleton
  @Named(AGENT_COORDINATOR_POLL)
  public ScheduledExecutorService agendCoordinatorPoll(SingleThreadFactoryBuilder threadFactoryBuilder) {
    return newSingleThreadScheduledExecutor(threadFactoryBuilder.build("agent-coordinator-poll"));
  }

  @Produces
  @Singleton
  public SingleThreadFactoryBuilder threadFactoryBuilder(FatalErrorHandler fatalErrorHandler) {
    return new SingleThreadFactoryBuilder(fatalErrorHandler);
  }

  public static class SingleThreadFactoryBuilder {

    private final FatalErrorHandler fatalErrorHandler;

    SingleThreadFactoryBuilder(FatalErrorHandler fatalErrorHandler) {
      this.fatalErrorHandler = fatalErrorHandler;
    }

    public ThreadFactory build(String threadName) {
      return runnable -> {
        Thread thread = Executors.defaultThreadFactory().newThread(runnable);
        thread.setName(threadName);
        thread.setDaemon(true);
        // The uncaught Exception handler should not be over-estimated.
        // It helps in some cases, but it is not always called when a scheduled task throws and Exception.
        // Example: Given an ExecutorService executor and a Runnable runnable that throws a RuntimeException:
        // - executor.execute(runnable) will trigger the uncaught Exception handler
        // - executor.submit(runnable) will not trigger the uncaught Exception handler
        // The reason is that submit returns a Future and the Exception is caught by the Future.
        thread.setUncaughtExceptionHandler((t, e) -> {
          LOGGER.error(e.getMessage(), e);
          fatalErrorHandler.systemExit(-1);
        });
        return thread;
      };
    }
  }
}
