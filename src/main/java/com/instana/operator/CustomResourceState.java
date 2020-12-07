package com.instana.operator;

import static com.instana.operator.client.KubernetesClientProducer.CRD_NAME;
import static com.instana.operator.util.ResourceUtils.name;
import static java.net.HttpURLConnection.HTTP_FORBIDDEN;

import java.util.Optional;

import javax.enterprise.context.ApplicationScoped;
import javax.inject.Inject;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import com.instana.operator.customresource.DoneableInstanaAgent;
import com.instana.operator.customresource.InstanaAgent;
import com.instana.operator.customresource.InstanaAgentList;
import com.instana.operator.customresource.InstanaAgentStatus;
import com.instana.operator.customresource.ResourceInfo;

import io.fabric8.kubernetes.api.model.HasMetadata;
import io.fabric8.kubernetes.api.model.Pod;
import io.fabric8.kubernetes.client.KubernetesClientException;
import io.fabric8.kubernetes.client.dsl.MixedOperation;
import io.fabric8.kubernetes.client.dsl.Resource;

@ApplicationScoped
public class CustomResourceState {

  @Inject
  FatalErrorHandler fatalErrorHandler;
  @Inject
  MixedOperation<InstanaAgent, InstanaAgentList, DoneableInstanaAgent, Resource<InstanaAgent, DoneableInstanaAgent>> client;

  private static final Logger LOGGER = LoggerFactory.getLogger(CustomResourceState.class);
  private InstanaAgent customResource = null;

  void customResourceAdded(InstanaAgent customResource) {
    if (this.customResource != null) {
      LOGGER.error("Illegal state: Custom resource " + name(customResource) + " was added, but custom resource " +
          name(this.customResource) + " already exists.");
      fatalErrorHandler.systemExit(-1);
    }
    this.customResource = customResource;
  }

  void customResourceDeleted() {
    customResource = null;
  }

  void customResourceModified(InstanaAgent modified) {
    if (customResource.getSpec().equals(modified.getSpec())) {
      // Spec has not changed, that's ok
      customResource = modified;
    } else {
      LOGGER.info("Custom resource " + CRD_NAME + " " + name(customResource) + " has been modified." +
          " The operator currently does not support updates. The changes will be discarded." +
          " Please delete and re-create the custom resource if you want to change the configuration.");
      update(); // reset changes
    }
  }

  <T extends HasMetadata> void update(T resource) {
    update(resource.getKind(), resource.getMetadata().getName(), resource.getMetadata().getUid());
  }

  void updateLeadingAgentPod(Pod pod) {
    Optional<InstanaAgentStatus> status = getStatus();
    if (!status.isPresent()) {
      return;
    }
    if (pod == null) {
      status.get().setLeadingAgentPod(null);
    } else {
      status.get().setLeadingAgentPod(new ResourceInfo(pod.getMetadata().getName(), pod.getMetadata().getUid()));
    }
    update();
  }

  void clearLeadingAgentPod() {
    updateLeadingAgentPod(null);
  }

  Optional<String> getLeadingAgentUid() {
    return getStatus()
        .map(InstanaAgentStatus::getLeadingAgentPod)
        .map(ResourceInfo::getUid);
  }

  private Optional<InstanaAgentStatus> getStatus() {
    if (customResource == null) {
      // This might happen if the custom resource was deleted, but the AgentLeaderManager
      // still has a scheduled task to select a new leader. This task will call getLeadingAgentUid()
      // although the customResource is null.
      return Optional.empty();
    }
    if (customResource.getStatus() == null) {
      customResource.setStatus(new InstanaAgentStatus());
    }
    return Optional.of(customResource.getStatus());
  }

  private void update(String kind, String name, String uid) {
    Optional<InstanaAgentStatus> status = getStatus();
    if (!status.isPresent()) {
      return;
    }
    switch (kind) {
    case "ServiceAccount":
      status.get().setServiceAccount(new ResourceInfo(name, uid));
      break;
    case "ClusterRole":
      status.get().setClusterRole(new ResourceInfo(name, uid));
      break;
    case "ClusterRoleBinding":
      status.get().setClusterRoleBinding(new ResourceInfo(name, uid));
      break;
    case "Secret":
      status.get().setSecret(new ResourceInfo(name, uid));
      break;
    case "ConfigMap":
      status.get().setConfigMap(new ResourceInfo(name, uid));
      break;
    case "DaemonSet":
      status.get().setDaemonSet(new ResourceInfo(name, uid));
      break;
    }
    update();
  }

  private void update() {
    try {
      client.inNamespace(customResource.getMetadata().getNamespace()).createOrReplace(customResource);
    } catch (Exception e) {
      StringBuilder errorMessage = new StringBuilder();
      errorMessage.append("Failed to update Custom Resource").append(CRD_NAME).append(name(customResource));
      if (e instanceof KubernetesClientException) {
        if (((KubernetesClientException)e).getCode() == HTTP_FORBIDDEN) {
          errorMessage.append(". Please ensure the operator has the updated cluster role permissions.");
        }
      }
      LOGGER.warn(errorMessage.toString() + ": " + e.getMessage());
      // No need to System.exit() if we cannot update the status. Ignore this and carry on.
    }
  }
}
