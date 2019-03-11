package com.instana.operator;

import io.fabric8.kubernetes.api.model.ConfigMap;
import io.fabric8.kubernetes.api.model.ObjectMeta;
import io.fabric8.kubernetes.api.model.OwnerReference;
import io.fabric8.kubernetes.api.model.Pod;
import io.fabric8.kubernetes.client.DefaultKubernetesClient;
import io.fabric8.kubernetes.client.KubernetesClient;

import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Paths;
import java.util.Collections;
import java.util.concurrent.TimeUnit;

// Java implementation of https://github.com/operator-framework/operator-sdk/blob/master/pkg/leader/leader.go
// If more than one instana-operator Pod is running, this is used to define which of these will be the leader.
// This is not the leader election for the Instana agent daemon set. It's leader election among multiple operator Pods.
// TODO: Check if Pod was restarted missing, see original implementation
class LeaderElector {

    private final KubernetesClient client;
    private final String lockName = "instana-operator-leader";

    static LeaderElector init() {
        KubernetesClient client = new DefaultKubernetesClient();
        return new LeaderElector(client);
    }

    private LeaderElector(KubernetesClient client) {
        this.client = client;
    }

    void becomeLeader() {
        ConfigMap configMap = createConfigMap();
        while (true) {
            try {
                client.configMaps().create(configMap);
                return;
            } catch (Exception e) {
                // another instance has already created the config map
                // poll every after 10 seconds
                try {
                    Thread.sleep(TimeUnit.SECONDS.toMillis(10));
                } catch (InterruptedException e1) {}
            }
        }
    }

    private ConfigMap createConfigMap() {
        OwnerReference ownerReference = myOwnerRef();
        ConfigMap configMap = new ConfigMap();
        ObjectMeta meta = new ObjectMeta();
        meta.setName(lockName);
        meta.setOwnerReferences(Collections.singletonList(ownerReference));
        configMap.setMetadata(meta);
        return configMap;
    }

    private OwnerReference myOwnerRef() {
        String operatorNamespace = findOperatorNamespace();
        String podName = System.getenv("POD_NAME");
        if (podName == null) {
            throw new RuntimeException("POD_NAME environment variable not set. Make sure to configure downward api in the deployment descriptor.");
        }
        Pod myself = client.pods().inNamespace(operatorNamespace).withName(podName).get();
        if (myself == null) {
            throw new RuntimeException("Failed to find the operator Pod. Name=" + podName + " namespace=" + operatorNamespace);
        }
        OwnerReference ownerReference = new OwnerReference();
        ownerReference.setApiVersion("v1");
        ownerReference.setKind("Pod");
        ownerReference.setName(myself.getMetadata().getName());
        ownerReference.setUid(myself.getMetadata().getUid());
        return ownerReference;
    }

    private String findOperatorNamespace() {
        try {
            byte[] bytes = Files.readAllBytes(Paths.get("/var/run/secrets/kubernetes.io/serviceaccount/namespace"));
            return new String(bytes, StandardCharsets.UTF_8).trim(); // TODO: encoding correct?
        } catch (IOException e) {
            throw new RuntimeException("namespace not found: this container is running outside of a Kubernetes cluster.");
        }
    }
}
