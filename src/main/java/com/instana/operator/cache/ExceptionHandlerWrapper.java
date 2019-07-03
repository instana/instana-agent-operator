package com.instana.operator.cache;

import com.instana.operator.service.FatalErrorHandler;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.function.BiConsumer;
import java.util.function.Consumer;

class ExceptionHandlerWrapper {

  private static final Logger LOGGER = LoggerFactory.getLogger(ExceptionHandlerWrapper.class);

  // Note: Wrapping the callbacks should not be necessary when using RxJava's subscribe() method,
  // because subscribe internally wraps onNext() and onError() in try-catch-blocks.
  // However, we leave it here in case somebody calls ListThenWatchOperation directly without
  // the reactive interface provided by CacheService.

  static <A, B> BiConsumer<A, B> exitOnError(BiConsumer<A, B> callback, FatalErrorHandler fatalErrorHandler) {
    return (a, b) -> {
      try {
        callback.accept(a, b);
      } catch (Exception e) {
        LOGGER.error(e.getMessage(), e);
        fatalErrorHandler.systemExit(-1);
      }
    };
  }

  static <A> Consumer<A> exitOnError(Consumer<A> callback, FatalErrorHandler fatalErrorHandler) {
    return (a) -> {
      try {
        callback.accept(a);
      } catch (Exception e) {
        LOGGER.error(e.getMessage(), e);
        fatalErrorHandler.systemExit(-1);
      }
    };
  }
}
