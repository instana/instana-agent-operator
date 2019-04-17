package com.instana.operator.service;

import static org.hamcrest.MatcherAssert.assertThat;
import static org.junit.jupiter.api.Assertions.assertNotNull;

import java.util.concurrent.TimeUnit;

import javax.inject.Inject;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import io.fabric8.kubernetes.api.model.OwnerReference;
import io.fabric8.kubernetes.api.model.Pod;
import io.fabric8.kubernetes.api.model.apps.Deployment;
import io.quarkus.test.junit.QuarkusTest;

@QuarkusTest
class OperatorOwnerReferenceServiceIT {

  @Inject
  KubernetesResourceService resourceService;
  @Inject
  OperatorNamespaceService namespaceService;
  @Inject
  OperatorOwnerReferenceService ownerRefService;

  Deployment operatorDeployment;
  Pod operatorPod;

  @BeforeEach
  void setUp() {
    operatorDeployment = resourceService.getKubernetesClient().apps().deployments()
        .inNamespace(namespaceService.getNamespace()).list()
        .getItems().stream()
        .filter(p -> p.getMetadata().getName().contains("instana-agent-operator"))
        .findFirst()
        .orElse(null);
    operatorPod = resourceService.getKubernetesClient().pods()
        .inNamespace(namespaceService.getNamespace()).list()
        .getItems().stream()
        .filter(p -> p.getMetadata().getName().contains("instana-agent-operator"))
        .findFirst()
        .orElse(null);
  }

  @Test
  void mustProvideDeploymentOwnerReference() throws Exception {
    assertNotNull(operatorDeployment);

    OwnerReference deployOwnerRef = ownerRefService.getOperatorDeploymentAsOwnerReference().get(5, TimeUnit.SECONDS);
    assertNotNull(deployOwnerRef);
    assertThat("UIDs match", deployOwnerRef.getUid().equals(operatorDeployment.getMetadata().getUid()));
  }

  @Test
  void mustProvidePodOwnerReference() throws Exception {
    assertNotNull(operatorPod);

    OwnerReference podOwnerRef = ownerRefService.getOperatorPodOwnerReference().get(5, TimeUnit.SECONDS);
    assertNotNull(podOwnerRef);
    assertThat("UIDs match", podOwnerRef.getUid().equals(operatorPod.getMetadata().getUid()));
  }

}
