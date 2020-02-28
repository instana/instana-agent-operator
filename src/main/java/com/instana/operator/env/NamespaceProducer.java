package com.instana.operator.env;

import com.instana.operator.FatalErrorHandler;
import com.instana.operator.util.StringUtils;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.inject.Produces;
import javax.inject.Inject;
import javax.inject.Named;
import javax.inject.Singleton;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Paths;
import java.util.Collections;
import java.util.Optional;
import java.util.Set;
import java.util.TreeSet;

import static com.instana.operator.env.Environment.OPERATOR_NAMESPACE;
import static com.instana.operator.env.Environment.TARGET_NAMESPACES;

@ApplicationScoped
public class NamespaceProducer {

  private static final Logger LOGGER = LoggerFactory.getLogger(NamespaceProducer.class);


  // From the CSV deployment spec:
  // ----------------------------------------------
  // - name: POD_NAME
  //   valueFrom:
  //     fieldRef:
  //       fieldPath: metadata.name
  // - name: OPERATOR_NAMESPACE
  //   valueFrom:
  //     fieldRef:
  //       fieldPath: metadata.namespace
  // - name: TARGET_NAMESPACES
  //   valueFrom:
  //     fieldRef:
  //       fieldPath: metadata.annotations['olm.targetNamespaces']

  @Inject
  FatalErrorHandler fatalErrorHandler;

  @Inject
  Environment environment;

  @Produces
  @Singleton
  @Named(OPERATOR_NAMESPACE)
  String findOperatorNamespace() {
    Optional<String> fromServiceaccount = loadOperatorNamespaceFromServiceaccount();
    Optional<String> fromEnv = loadOperatorNamespaceFromEnv();

    if (fromServiceaccount.isPresent() && fromEnv.isPresent()) {
      if (!fromServiceaccount.get().equals(fromEnv.get())) {
        LOGGER.error("Deployment error: The agent pod runs in namespace " + fromServiceaccount.get() + " but the" +
            " environment variable " + OPERATOR_NAMESPACE + " is set to " + fromEnv.get() + ".");
        fatalErrorHandler.systemExit(-1);
      }
    }
    if (fromServiceaccount.isPresent()) {
      return fromServiceaccount.get();
    }
    if (fromEnv.isPresent()) {
      return fromEnv.get();
    }
    LOGGER.error(
        "Failed to find the namespace where the operator is running." +
            " If you run the operator outside of a Kubernetes cluster, make sure to configure the " +
            OPERATOR_NAMESPACE + " environment variable via Downward API.");
    fatalErrorHandler.systemExit(-1);
    return null; // will not happen, because we called System.exit()
  }

  @Produces
  @Singleton
  @Named(TARGET_NAMESPACES)
  Set<String> loadTargetNamespaces() {
    Set<String> result = new TreeSet<>();
    String targetNamespaces = environment.get(TARGET_NAMESPACES);
    if (!StringUtils.isBlank(targetNamespaces)) {
      for (String ns : targetNamespaces.split(",")) {
        result.add(ns.trim());
      }
    }
    return Collections.unmodifiableSet(result);
  }

  private Optional<String> loadOperatorNamespaceFromServiceaccount() {
    try {
      byte[] bytes = Files.readAllBytes(Paths.get("/var/run/secrets/kubernetes.io/serviceaccount/namespace"));
      return Optional.of(new String(bytes));
    } catch (IOException e) {
      return Optional.empty();
    }
  }

  private Optional<String> loadOperatorNamespaceFromEnv() {
    return Optional.ofNullable(environment.get(OPERATOR_NAMESPACE)).filter(s -> !StringUtils.isBlank(s));
  }
}
