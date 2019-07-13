package com.instana.operator.env;

import com.instana.operator.FatalErrorHandler;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.inject.Produces;
import javax.inject.Inject;
import javax.inject.Named;
import javax.inject.Singleton;

@ApplicationScoped
public class OperatorPodNameProducer {

  private static final Logger LOGGER = LoggerFactory.getLogger(OperatorPodNameProducer.class);
  public static final String POD_NAME = "POD_NAME";

  @Inject
  FatalErrorHandler fatalErrorHandler;

  @Produces
  @Singleton
  @Named(POD_NAME)
  String findPodName() {
    String podName = System.getenv(POD_NAME);
    if (podName == null) {
      LOGGER.error("Environment variable " + POD_NAME + " not found." +
          " Please ensure the Downward API for " + POD_NAME + " is set using the provided YAML.");
      fatalErrorHandler.systemExit(-1);
    }
    return podName;
  }
}
