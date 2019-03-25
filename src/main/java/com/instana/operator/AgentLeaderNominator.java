package com.instana.operator;

import io.fabric8.kubernetes.api.model.ContainerStatus;
import io.fabric8.kubernetes.api.model.OwnerReference;
import io.fabric8.kubernetes.api.model.Pod;
import io.fabric8.kubernetes.api.model.PodList;
import io.fabric8.kubernetes.client.KubernetesClient;
import io.fabric8.kubernetes.client.KubernetesClientException;
import io.fabric8.kubernetes.client.Watch;
import io.fabric8.kubernetes.client.Watcher;
import io.reactivex.Observable;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.Collections;
import java.util.List;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;
import java.util.function.Predicate;
import java.util.stream.Collectors;

class AgentLeaderNominator {

    private static final String DAEMON_SET_NAME = "instana-agent";
    private static final Logger logger = LoggerFactory.getLogger(AgentLeaderNominator.class);

    private final KubernetesClient client;
    private final String namespace;

    private static List<Pod> allAgents;
    private static Pod leader;

    AgentLeaderNominator(KubernetesClient client, String namespace) {
        this.client = client;
        this.namespace = namespace;
    }

    // TODO: rx.Observable can be replaced with any other implementation.
    Observable<Pod> nominationLoop() {
        return Observable.create(subscriber -> {
            try {
                while (true) {
                    List<Pod> newAgentList = findPods(belongsToInstanaAgentDaemonSet.and(isRunning));
                    if (newAgentList.isEmpty()) {
                        subscriber.onError(new InitializationException("No instana agent found in namespace " + namespace + "."));
                    }
                    if (allAgents == null || !hasSamePods(allAgents, newAgentList)) {
                        allAgents = newAgentList;
                        logger.info("Updated list of instana-agent pods in namespace " + namespace + ": " + podListToString(allAgents));
                        if (leader == null || !contains(allAgents, leader)) {
                            if (leader != null) {
                                logger.info("Current instana-agent leader in namespace " + namespace + " no longer available: " + leader.getMetadata().getName());
                            }
                            leader = allAgents.get(0); // TODO: Should we use random here, or just take the 1st?
                            logger.info("Nominating new instana-agent leader in namespace " + namespace + ": " + leader.getMetadata().getName());
                            subscriber.onNext(leader);
                        }
                    }
                    waitForPodEvent(10, TimeUnit.MINUTES); // Stop the watch and list all Pods every 10 minutes.
                }
            } catch (Exception e) {
                subscriber.onError(e);
            }
        });
    }

    void waitForPodEvent(long timeout, TimeUnit timeUnit) throws InterruptedException {
        CountDownLatch lock = new CountDownLatch(1);
        Watch watch = client.pods().inNamespace(namespace).watch(
                new Watcher<Pod>() {
                    @Override
                    public void eventReceived(Action action, Pod resource) {
                        String event = "" + action + " event for Pod " + resource.getMetadata().getName();
                        if (! canBeIgnored(action, resource, event)) {
                            logger.info(event + " will trigger re-scanning of the Pod list.");
                            lock.countDown();
                        }
                    }

                    @Override
                    public void onClose(KubernetesClientException cause) {
                        lock.countDown();
                    }
                }
        );
        long timestamp = System.nanoTime();
        lock.await(timeout, timeUnit);
        logger.debug("Re-scanning the Pod list after " + (TimeUnit.NANOSECONDS.toSeconds(System.nanoTime()-timestamp)) + " seconds.");
        watch.close();
    }

    private boolean canBeIgnored(Watcher.Action action, Pod resource, String event) {
        if (! belongsToInstanaAgentDaemonSet.test(resource)) {
            logger.debug("Ignoring " + event + ", because this Pod does not belong to the " + DAEMON_SET_NAME + " daemon set.");
            return true;
        }
        switch (action) {
            case ADDED:
                if (contains(allAgents, resource)) {
                    logger.debug("Ignoring " + event + ", because this Pod is already in the list of known Pods.");
                    return true;
                }
            case MODIFIED:
                if (contains(allAgents, resource, true)) {
                    logger.debug("Ignoring " + event + ", because the current generation of this Pod is already in the list of known Pods.");
                    return true;
                }
            case ERROR:
                // fall through
            case DELETED:
                if (!contains(allAgents, resource)) {
                    logger.debug("Ignoring " + event + ", because this Pod is not in the list of known Pods.");
                    return true;
                }
        }
        return false;
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

    private List<Pod> findPods(Predicate<Pod> filter) {
        PodList podList = client.pods().inNamespace(namespace).list();
        if (podList == null || podList.getItems() == null) {
            return Collections.emptyList();
        }
        return podList.getItems().stream().filter(filter).collect(Collectors.toList());
    }

    private static boolean contains(List<Pod> pods, Pod pod) {
        return contains(pods, pod, false);
    }

    private static boolean contains(List<Pod> pods, Pod pod, boolean compareGeneration) {
        for (Pod p : pods) {
            if (p.getMetadata().getName().equals(pod.getMetadata().getName())) {
                if (compareGeneration && p.getMetadata().getGeneration() != null && p.getMetadata().getGeneration().equals(pod.getMetadata().getGeneration())) {
                    return true;
                }
                if (! compareGeneration) {
                    return true;
                }
            }
        }
        return false;
    }

    private static boolean hasSamePods(List<Pod> a, List<Pod> b) {
        if (a.size() != b.size()) {
            return false;
        }
        for (Pod aPod : a) {
            if (!contains(b, aPod, true)) {
                return false;
            }
        }
        return true;
    }

    private static String podListToString(List<Pod> podList) {
        return podList.stream()
                .map(agent -> agent.getMetadata().getName())
                .reduce((a, b) -> a + ", " + b)
                .get();
    }
}
