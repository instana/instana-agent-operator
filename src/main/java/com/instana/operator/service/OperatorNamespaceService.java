package com.instana.operator.service;

import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.event.Event;
import javax.inject.Inject;

import com.instana.operator.GlobalErrorEvent;

import io.reactivex.Maybe;
import io.reactivex.Single;

/**
 * Service that is responsible for looking up and providing the namespace the Operator
 * is deployed into, as well as the Pod name of the Operator.
 */
@ApplicationScoped
public class OperatorNamespaceService {

  private static final String POD_NAME = "POD_NAME";
  private static final String POD_NAMESPACE = "POD_NAMESPACE";

  @Inject
  Event<GlobalErrorEvent> globalErrorEvent;

  private String name;
  private String namespace;

  /**
   * Get the pod name of the Operator, as passed into the environment via the Downward API.
   *
   * @return operator pod name
   */
  synchronized String getOperatorPodName() {
    if (null == name && null == (name = System.getenv(POD_NAME))) {
      globalErrorEvent.fire(new GlobalErrorEvent(new IllegalStateException(
          "POD_NAME not available in the environment. "
              + "Please ensure the downward API for POD_NAME is set using the provided YAML.")));
      return null; // NPE
    }
    return name;
  }

  /**
   * Get the namespace the Operator is deployed into.
   *
   * @return operator namespace
   */
  public synchronized String getNamespace() {
    if (null == namespace) {
      namespace = findNamespaceWithServiceAccount()
          .flatMap(OperatorNamespaceService::readFile)
          .switchIfEmpty(findNamespaceWithEnvironmentVariable())
          .blockingGet();
    }
    return namespace;
  }

  private Single<String> findNamespaceWithEnvironmentVariable() {
    String ns = System.getenv(POD_NAMESPACE);
    if (null == ns) {
      globalErrorEvent.fire(new GlobalErrorEvent(new IllegalStateException(
          "POD_NAME not available in the environment. "
              + "Please ensure the downward API for POD_NAME is set using the provided YAML.")));
      return null; // NPE
    }
    return Single.just(ns);
  }

  private static Maybe<Path> findNamespaceWithServiceAccount() {
    return Single.just(Paths.get("/var/run/secrets/kubernetes.io/serviceaccount/namespace"))
        .filter(Files::exists);
  }

  private static Maybe<String> readFile(Path p) {
    try {
      byte[] bytes = Files.readAllBytes(p);
      return Maybe.just(new String(bytes, StandardCharsets.UTF_8).trim());
    } catch (IOException e) {
      return Maybe.error(e);
    }
  }

}

