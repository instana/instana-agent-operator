/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc.
 */
package com.instana.operator.env;

import com.instana.operator.FatalErrorHandler;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.inject.Produces;
import javax.inject.Inject;
import javax.inject.Named;
import javax.inject.Singleton;

import static com.instana.operator.env.Environment.POD_NAME;

@ApplicationScoped
public class OperatorPodNameProducer {

  private static final Logger LOGGER = LoggerFactory.getLogger(OperatorPodNameProducer.class);

  @Inject
  FatalErrorHandler fatalErrorHandler;

  @Inject
  Environment environment;

  @Produces
  @Singleton
  @Named(POD_NAME)
  String findPodName() {
    String podName = environment.get(POD_NAME);
    if (podName == null) {
      LOGGER.error("Environment variable " + POD_NAME + " not found." +
          " Please ensure the Downward API for " + POD_NAME + " is set using the provided YAML.");
      fatalErrorHandler.systemExit(-1);
    }
    return podName;
  }
}
