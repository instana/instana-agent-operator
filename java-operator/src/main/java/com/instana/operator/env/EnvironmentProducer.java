/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc.
 */
package com.instana.operator.env;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.inject.Produces;
import javax.inject.Singleton;

@ApplicationScoped
public class EnvironmentProducer {
  @Produces
  @Singleton
  Environment getEnvironment() {
    return Environment.fromMap(System.getenv());
  }
}
