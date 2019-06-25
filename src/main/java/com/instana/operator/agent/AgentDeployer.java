package com.instana.operator.agent;

import com.instana.operator.GlobalErrorEvent;
import com.instana.operator.customresource.InstanaAgentSpec;
import com.instana.operator.service.AgentConfigFoundEvent;
import com.instana.operator.service.KubernetesResourceService;
import com.instana.operator.service.OperatorNamespaceService;
import com.instana.operator.service.OperatorOwnerReferenceService;
import io.fabric8.kubernetes.api.model.*;
import io.fabric8.kubernetes.api.model.apps.DaemonSet;
import io.fabric8.kubernetes.api.model.rbac.ClusterRole;
import io.fabric8.kubernetes.api.model.rbac.ClusterRoleBinding;
import io.fabric8.kubernetes.client.KubernetesClientException;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.event.Event;
import javax.enterprise.event.ObservesAsync;
import javax.inject.Inject;
import java.nio.charset.Charset;
import java.util.Base64;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.TimeoutException;

import static com.instana.operator.util.AgentResourcesUtil.*;

@ApplicationScoped
public class AgentDeployer {

  private static final Logger LOGGER = LoggerFactory.getLogger(AgentDeployer.class);

  @Inject
  KubernetesResourceService clientService;
  @Inject
  OperatorNamespaceService namespaceService;
  @Inject
  OperatorOwnerReferenceService ownerReferenceService;
  @Inject
  Event<GlobalErrorEvent> globalErrorEvent;

  void onAgentConfigFound(@ObservesAsync AgentConfigFoundEvent ev) {

    String namespace = namespaceService.getNamespace();
    InstanaAgentSpec config = ev.getConfig().getSpec();

    LOGGER.debug("Finding the operator deployment as owner reference...");
    OwnerReference ownerRef = null;
    try {
      ownerRef = ownerReferenceService.getOperatorDeploymentAsOwnerReference().get(30, TimeUnit.SECONDS);
    } catch (InterruptedException | ExecutionException | TimeoutException e) {
      globalErrorEvent.fire(new GlobalErrorEvent("Timeout while getting operator deployment as owner reference", e));
    }

    ServiceAccount serviceAccount = createServiceAccount(
        namespace, config.getAgentServiceAccountName(), ownerRef);
    maybeCreateResource(serviceAccount);

    if (config.isAgentRbacCreate()) {
      ClusterRole clusterRole = createAgentClusterRole(
          config.getAgentClusterRoleName(), ownerRef);
      maybeCreateResource(clusterRole);

      ClusterRoleBinding clusterRoleBinding = createAgentClusterRoleBinding(
          namespace, config.getAgentClusterRoleBindingName(), serviceAccount, clusterRole, ownerRef);
      maybeCreateResource(clusterRoleBinding);
    }

    Secret secret = createAgentKeySecret(
        namespace, config.getAgentSecretName(), base64(config.getAgentKey()), ownerRef);
    maybeCreateResource(secret);

    ConfigMap agentConfigMap = createConfigurationConfigMap(namespace, config.getAgentConfigMapName(),
        config.getConfigFiles(), ownerRef);
    maybeCreateResource(agentConfigMap);

    DaemonSet daemonSet = createAgentDaemonSet(
        namespace,
        config.getAgentDaemonSetName(),
        serviceAccount,
        secret,
        agentConfigMap,
        ownerRef,
        config.getAgentDownloadKey(),
        config.getAgentZoneName(),
        config.getAgentEndpointHost(),
        config.getAgentEndpointPort(),
        config.getAgentMode(),
        config.getAgentCpuReq(),
        config.getAgentMemReq(),
        config.getAgentCpuLimit(),
        config.getAgentMemLimit(),
        config.getAgentImageName(),
        config.getAgentImageTag(),
        config.getAgentProxyHost(),
        config.getAgentProxyPort(),
        config.getAgentProxyProtocol(),
        config.getAgentProxyUser(),
        config.getAgentProxyPassword(),
        config.isAgentProxyUseDNS(),
        config.getAgentHttpListen());
    maybeCreateResource(daemonSet);
  }

  @SuppressWarnings("unchecked")
  private <T extends HasMetadata> void maybeCreateResource(T resource) {
    LOGGER.debug("Creating {} at {}/{}...",
        resource.getKind(),
        namespaceService.getNamespace(),
        resource.getMetadata().getName());
    try {
      clientService.getKubernetesClient().create(namespaceService.getNamespace(), resource);
    } catch (KubernetesClientException e) {

      if (e.getCode() != 409) {
        if (e.getCause() instanceof java.io.InterruptedIOException) {
          LOGGER.error(
              "Thread interrupted while creating " + resource.getKind() + "/" + resource.getMetadata().getName()
                  + " in namespace " + namespaceService.getNamespace());
        } else {
          globalErrorEvent.fire(new GlobalErrorEvent(
              "Kubernetes API server responded HTTP code " + e.getCode() + " while creating " + resource.getClass()
                  .getSimpleName(), e.getCause()));
        }
      } else {
        LOGGER.info("{} {}/{} already exists.",
            resource.getKind(),
            namespaceService.getNamespace(),
            resource.getMetadata().getName());
      }
    }
  }

  private String base64(String secret) {
    return new String(Base64.getEncoder().encode(secret.getBytes(Charset.forName("ASCII"))), Charset.forName("ASCII"));
  }

}
