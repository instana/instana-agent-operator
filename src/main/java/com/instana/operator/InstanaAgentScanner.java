package com.instana.operator;

import io.fabric8.kubernetes.api.model.ContainerStatus;
import io.fabric8.kubernetes.api.model.OwnerReference;
import io.fabric8.kubernetes.api.model.Pod;
import io.fabric8.kubernetes.api.model.PodList;
import io.fabric8.kubernetes.client.KubernetesClient;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.Collections;
import java.util.List;
import java.util.function.Predicate;
import java.util.stream.Collectors;

import static com.instana.operator.KubernetesUtil.findNamespace;

class InstanaAgentScanner {

    private final Logger logger = LoggerFactory.getLogger(InstanaAgentScanner.class);
    private final KubernetesClient client;

    private final String DAEMON_SET_NAME = "instana-agent"; // must be the same as in the instana-agent's deployment descriptor

    InstanaAgentScanner(KubernetesClient client) {
        this.client = client;
    }

    private Predicate<Pod> belongsToInstanaAgentDaemonSet = pod -> {
        if (pod.getMetadata() != null && pod.getMetadata().getOwnerReferences() != null) {
            for (OwnerReference ownerReference : pod.getMetadata().getOwnerReferences()) {
                if ("DaemonSet".equals(ownerReference.getKind()) && DAEMON_SET_NAME.equals(ownerReference.getName())) {
                    return true;
                }
            }
        }
        return false;
    };

    private Predicate<Pod> isRunning = pod -> {
        if (pod.getStatus() != null && pod.getStatus().getContainerStatuses() != null) {
            for (ContainerStatus containerStatus : pod.getStatus().getContainerStatuses()) {
                if (containerStatus.getState() != null) {
                    if (containerStatus.getState().getRunning() == null) {
                        return false;
                    }
                }
            }
        }
        return true;
    };

    void run() throws InitializationException {
        String namespace = findNamespace();
        List<Pod> instanaAgents = findPods(namespace, belongsToInstanaAgentDaemonSet.and(isRunning));
        if (instanaAgents.isEmpty()) {
            logger.info("did not find any instana-agent in namespace " + namespace);
            return;
        }
        String agentList = instanaAgents.stream()
                .map(agent -> agent.getMetadata().getName())
                .reduce((a, b) -> a + ", " + b)
                .get();
        logger.debug("found " + instanaAgents.size() + " instana agents: " + agentList);
    }

    private List<Pod> findPods(String namespace, Predicate<Pod> filter) throws InitializationException {
        PodList podList = client.pods().inNamespace(namespace).list();
        if (podList == null || podList.getItems() == null) {
            return Collections.emptyList();
        }
        return podList.getItems().stream().filter(filter).collect(Collectors.toList());
    }
}
