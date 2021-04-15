/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc.
 */
package com.instana.operator;

import com.instana.operator.env.Environment;
import com.instana.operator.events.OperatorPodRunning;
import com.instana.operator.events.OperatorPodValidated;
import com.instana.operator.util.StringUtils;
import io.fabric8.kubernetes.api.model.Pod;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.event.Event;
import javax.enterprise.event.NotificationOptions;
import javax.enterprise.event.ObservesAsync;
import javax.inject.Inject;
import javax.inject.Named;
import java.util.Set;
import java.util.TreeSet;

import static com.instana.operator.env.Environment.OPERATOR_NAMESPACE;
import static com.instana.operator.env.Environment.POD_NAME;
import static com.instana.operator.env.Environment.TARGET_NAMESPACES;

@ApplicationScoped
public class OperatorPodValidator {

  private static final Logger LOGGER = LoggerFactory.getLogger(OperatorPodValidator.class);

  @Inject
  @Named(POD_NAME)
  String podName;
  @Inject
  @Named(OPERATOR_NAMESPACE)
  String operatorNamespace;
  @Inject
  @Named(TARGET_NAMESPACES)
  Set<String> targetNamespaces;
  @Inject
  FatalErrorHandler fatalErrorHandler;
  @Inject
  Event<OperatorPodValidated> operatorPodValidatedEventEvent;
  @Inject
  NotificationOptions asyncSerial;
  @Inject
  Environment environment;

  void onOperatorPodRunning(@ObservesAsync OperatorPodRunning event) {
    Pod myself = event.getOperatorPod();
    validate(myself);
    LOGGER.debug("Annotations for Pod " + operatorNamespace + "/" + podName + " are valid.");
    operatorPodValidatedEventEvent.fireAsync(new OperatorPodValidated(myself), asyncSerial)
        .exceptionally(fatalErrorHandler::logAndExit);
  }

  // If the operator is started using the Operator Lifecycle Manager (OLM),
  // the Pod will be annotated with olm.* annotations.
  // If these annotations are present, make sure they are consistent with our configuration.
  private void validate(Pod myself) {
    if (myself.getMetadata().getAnnotations() != null) {
      String olmOperatorNamespace = myself.getMetadata().getAnnotations().get("olm.operatorNamespace");
      if (!StringUtils.isBlank(olmOperatorNamespace)) {
        if (!olmOperatorNamespace.equals(operatorNamespace)) {
          LOGGER.error("Configuration error: The operator Pod " + podName + " runs in namespace " + operatorNamespace
              + " but is annotated with olm.operatorNamespace: " + olmOperatorNamespace);
          fatalErrorHandler.systemExit(-1);
        }
      }
      String olmTargetNamespaces = myself.getMetadata().getAnnotations().get("olm.targetNamespaces");
      if (!StringUtils.isBlank(olmTargetNamespaces)) {
        if (environment.get(TARGET_NAMESPACES) == null) {
          LOGGER.error("Configuration error: The operator Pod " + podName + " is annotated with" +
              " olm.targetNamespaces: " + olmTargetNamespaces + " but the environment variable " + TARGET_NAMESPACES +
              " is empty.");
          fatalErrorHandler.systemExit(-1);
        }
        TreeSet<String> nss = new TreeSet<>();
        for (String ns : olmOperatorNamespace.split(",")) {
          nss.add(ns.trim());
        }
        if (!targetNamespaces.equals(nss)) {
          LOGGER.error("Configuration error: The operator Pod " + podName + " is annotated with" +
              " olm.targetNamespaces: " + olmTargetNamespaces + " but has environment variable " + TARGET_NAMESPACES +
              "=\"" + environment.get(TARGET_NAMESPACES) + "\".");
          fatalErrorHandler.systemExit(-1);
        }
      }
    }
  }
}
