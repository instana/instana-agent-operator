package com.instana.operator;

import io.fabric8.kubernetes.client.DefaultKubernetesClient;
import io.fabric8.kubernetes.client.KubernetesClient;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Paths;

public class Main {

    private static final Logger logger = LoggerFactory.getLogger(Main.class);

    public static void main(String[] args) throws InitializationException {
        KubernetesClient client = new DefaultKubernetesClient();
        String namespace = "";
        OperatorLeaderElector operatorLeaderElector = new OperatorLeaderElector(client, namespace);
        operatorLeaderElector.waitUntilBecomingLeader();
        AgentLeaderNominator agentNominator = new AgentLeaderNominator(client, namespace);
        agentNominator.nominationLoop().subscribe(
                leader -> {
                    logger.info(leader.getMetadata().getName() + " was nominated as new leader.");
                    // TODO: At this point we should inform the leader of its nomination.
                },
                ex -> {
                    logger.error(ex.getMessage(), ex);
                    client.close();
                },
                () -> {
                    client.close();
                }
        );
    }

}
