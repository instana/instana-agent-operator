package com.instana.operator.agent;

import static com.instana.operator.util.AgentResourcesUtil.createAgentClusterRole;
import static com.instana.operator.util.AgentResourcesUtil.createAgentClusterRoleBinding;
import static com.instana.operator.util.AgentResourcesUtil.createAgentDaemonSet;
import static com.instana.operator.util.AgentResourcesUtil.createAgentKeySecret;
import static com.instana.operator.util.AgentResourcesUtil.createServiceAccount;

import java.nio.charset.Charset;
import java.util.Base64;
import java.util.concurrent.ScheduledExecutorService;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.event.Event;
import javax.enterprise.event.Observes;
import javax.inject.Inject;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import com.instana.operator.GlobalErrorEvent;
import com.instana.operator.config.InstanaConfig;
import com.instana.operator.leaderelection.LeaderElectionEvent;
import com.instana.operator.service.KubernetesResourceService;
import com.instana.operator.service.OperatorNamespaceService;
import com.instana.operator.service.OperatorOwnerReferenceService;

import io.fabric8.kubernetes.api.model.ConfigMap;
import io.fabric8.kubernetes.api.model.HasMetadata;
import io.fabric8.kubernetes.api.model.Secret;
import io.fabric8.kubernetes.api.model.ServiceAccount;
import io.fabric8.kubernetes.api.model.apps.DaemonSet;
import io.fabric8.kubernetes.api.model.rbac.ClusterRole;
import io.fabric8.kubernetes.api.model.rbac.ClusterRoleBinding;
import io.fabric8.kubernetes.client.KubernetesClientException;
import io.fabric8.kubernetes.client.NamespacedKubernetesClient;
import io.fabric8.kubernetes.client.dsl.MixedOperation;

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
  ScheduledExecutorService executorService;
  @Inject
  InstanaConfig instanaConfig;
  @Inject
  Event<GlobalErrorEvent> globalErrorEvent;

  void onLeaderElection(@Observes LeaderElectionEvent ev) {
    if (!ev.isLeader()) {
      return;
    }

    String namespace = namespaceService.getNamespace();
    NamespacedKubernetesClient client = clientService.getKubernetesClient();

    LOGGER.debug("Finding the operator deployment as owner reference...");
    ownerReferenceService.getOperatorDeploymentAsOwnerReference()
        .thenAcceptAsync(ownerRef -> {

          ServiceAccount serviceAccount = createServiceAccount(
              namespace, instanaConfig.getServiceAccountName(), ownerRef);
          maybeCreateResource(serviceAccount, client.serviceAccounts());

          if (instanaConfig.isRbacCreate()) {
            ClusterRole clusterRole = createAgentClusterRole(
                instanaConfig.getClusterRoleName(), ownerRef);
            maybeCreateResource(clusterRole, client.rbac().clusterRoles());

            ClusterRoleBinding clusterRoleBinding = createAgentClusterRoleBinding(
                namespace, instanaConfig.getClusterRoleBindingName(), serviceAccount, clusterRole, ownerRef);
            maybeCreateResource(clusterRoleBinding, client.rbac().clusterRoleBindings());
          }

          Secret secret = createAgentKeySecret(
              namespace, instanaConfig.getSecretName(), base64(instanaConfig.getAgentKey()), ownerRef);
          maybeCreateResource(secret, client.secrets());

          ConfigMap agentConfigMap = clientService.getKubernetesClient().configMaps()
              .inNamespace(namespaceService.getNamespace())
              .withName(instanaConfig.getConfigMapName())
              .get();
          if (null == agentConfigMap) {
            globalErrorEvent.fire(new GlobalErrorEvent(new IllegalStateException(
                "Agent ConfigMap named " + instanaConfig.getConfigMapName() + " not found in namespace "
                    + namespaceService.getNamespace())));
            return;
          }

          DaemonSet daemonSet = createAgentDaemonSet(
              namespace,
              instanaConfig.getDaemonSetName(),
              serviceAccount,
              secret,
              agentConfigMap,
              ownerRef,
              instanaConfig.getZoneName(),
              instanaConfig.getEndpoint(),
              instanaConfig.getEndpointPort(),
              instanaConfig.getAgentCpuReq(),
              instanaConfig.getAgentMemReq(),
              instanaConfig.getAgentCpuLimit(),
              instanaConfig.getAgentMemLimit(),
              instanaConfig.getAgentImageName(),
              instanaConfig.getAgentImageTag(),
              instanaConfig.getAgentProxyHost(),
              instanaConfig.getAgentProxyPort(),
              instanaConfig.getAgentProxyProtocol(),
              instanaConfig.getAgentProxyUser(),
              instanaConfig.getAgentProxyPasswd(),
              instanaConfig.getAgentProxyUseDNS(),
              instanaConfig.getAgentHttpListen());
          maybeCreateResource(daemonSet, client.apps().daemonSets());

          LOGGER.debug("Successfully deployed the Instana agent.");
        }, executorService);
  }

  @SuppressWarnings("unchecked")
  private <T extends HasMetadata> void maybeCreateResource(T resource, MixedOperation<T, ?, ?, ?> op) {
    LOGGER.debug("Creating {} at {}/{}...",
        resource.getKind(),
        namespaceService.getNamespace(),
        resource.getMetadata().getName());
    try {
      op.inNamespace(namespaceService.getNamespace()).create(resource);
    } catch (KubernetesClientException e) {
      if (e.getCode() != 409) {
        globalErrorEvent.fire(new GlobalErrorEvent(e.getCause()));
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
