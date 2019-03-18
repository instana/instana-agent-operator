package com.instana.operator;

import io.fabric8.kubernetes.api.model.ConfigMap;
import io.fabric8.kubernetes.api.model.ObjectMeta;
import io.fabric8.kubernetes.api.model.OwnerReference;
import io.fabric8.kubernetes.api.model.Pod;
import io.fabric8.kubernetes.client.KubernetesClient;
import io.fabric8.kubernetes.client.KubernetesClientException;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.Collections;
import java.util.concurrent.TimeUnit;

// Java implementation of https://github.com/operator-framework/operator-sdk/blob/master/pkg/leader/leader.go
// If more than one instana-operator Pod is running, this is used to define which of these will be the leader.
class OperatorLeaderElector {

    private final Logger logger = LoggerFactory.getLogger(OperatorLeaderElector.class);

    private final KubernetesClient client;
    private final String namespace;
    private final String lockName = "instana-operator-leader-lock";
    private final int pollIntervalSeconds = 10; // TODO: Short interval is good for development but should be adjusted for production.

    OperatorLeaderElector(KubernetesClient client, String namespace) {
        this.namespace = namespace;
        this.client = client;
    }

    // TODO: If the Java application is restarted within a running operator Pod (e.g. the update script updated the
    // Java application without restarting the Pod), there might be an existing ConfigMap with the current Pod as Owner.
    // Detect this and assume this Pod is the leader if that happens.
    void waitUntilBecomingLeader() throws InitializationException {
        boolean firstTry = true;
        ConfigMap configMap = createConfigMap();
        while (true) {
            try {
                client.configMaps().create(configMap);
                logger.info("Leader election: Successfully became the leader.");
                return;
            } catch (KubernetesClientException e) {
                // For status codes, see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#http-status-codes
                if (e.getCode() != 409) {
                    throw new InitializationException("Failed to create config map: " + e.getMessage(), e);
                }
                if (firstTry) {
                    logger.info("Leader election: Another Instana operator instance is currently leader. Entering poll loop and trying to become leader every " + pollIntervalSeconds + " seconds.");
                    firstTry = false;
                }
                try {
                    Thread.sleep(TimeUnit.SECONDS.toMillis(pollIntervalSeconds));
                } catch (InterruptedException i) {
                    throw new InitializationException("Thread was interrupted while waiting to become leader.", i);
                }
            }
        }
    }

    private ConfigMap createConfigMap() throws InitializationException {
        OwnerReference ownerReference = findMyOwnerRef();
        ConfigMap configMap = new ConfigMap();
        ObjectMeta meta = new ObjectMeta();
        meta.setName(lockName);
        meta.setOwnerReferences(Collections.singletonList(ownerReference));
        configMap.setMetadata(meta);
        return configMap;
    }

    private OwnerReference findMyOwnerRef() throws InitializationException {
        String podName = System.getenv("POD_NAME");
        if (podName == null) {
            throw new InitializationException("POD_NAME environment variable not set. Make sure to configure downward API in the deployment descriptor.");
        }
        String errMsg = "Failed to find Pod '" + podName + "' in namespace '" + namespace + "'";
        try {
            Pod myself = client.pods().inNamespace(namespace).withName(podName).get();
            if (myself == null) {
                throw new InitializationException(errMsg + ".");
            }
            OwnerReference ownerReference = new OwnerReference();
            ownerReference.setApiVersion("v1");
            ownerReference.setKind("Pod");
            ownerReference.setName(myself.getMetadata().getName());
            ownerReference.setUid(myself.getMetadata().getUid());
            return ownerReference;
        } catch (KubernetesClientException e) {
            throw new InitializationException(errMsg + ": " + e.getMessage(), e);
        }
    }
}
