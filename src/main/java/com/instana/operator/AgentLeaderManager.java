package com.instana.operator;

import com.instana.operator.events.AgentPodAdded;
import com.instana.operator.events.AgentPodDeleted;
import io.fabric8.kubernetes.api.model.Pod;
import okhttp3.MediaType;
import okhttp3.OkHttpClient;
import okhttp3.Request;
import okhttp3.RequestBody;
import okhttp3.Response;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.event.ObservesAsync;
import javax.inject.Inject;
import javax.inject.Named;
import java.io.IOException;
import java.util.HashMap;
import java.util.HashSet;
import java.util.Map;
import java.util.Optional;
import java.util.Random;
import java.util.Set;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;

import static com.instana.operator.ExecutorProducer.CDI_HANDLER;
import static com.instana.operator.client.KubernetesClientProducer.AGENT_POD_HTTP_CLIENT;
import static com.instana.operator.util.ResourceUtils.name;

@ApplicationScoped
public class AgentLeaderManager {

  private static final Logger LOGGER = LoggerFactory.getLogger(AgentLeaderManager.class);
  private static final int AGENT_PORT = 42699;

  @Inject
  @Named(AGENT_POD_HTTP_CLIENT)
  OkHttpClient httpClient;

  @Inject
  KubernetesEventService kubernetesEventService;
  @Inject
  CustomResourceState customResourceState;
  @Inject
  @Named(CDI_HANDLER)
  ScheduledExecutorService executor;

  private final Random random = new Random();
  private final Map<String, Pod> agentPods = new HashMap<>();
  private String leaderUID = null;
  private final Set<String> nonLeadersToBeNotified = new HashSet<>();

  // All methods are called in the cdi-handler thread, thus the
  // implementation does not need to be thread save.

  public void onAgentPodAdded(@ObservesAsync AgentPodAdded event) {
    LOGGER.info("Found agent Pod " + name(event.getPod()) + ".");
    agentPods.put(event.getPod().getMetadata().getUid(), event.getPod());

    // Don't set any leader status right away, because if the DaemonSet is starting
    // up or shutting down we will get a lot of errors and re-tries while agent Pods are not yet ready
    // or while they are being deleted. Schedule the next steps in a few seconds to give it some slack.

    // We can safely schedule a "not leader" notification here, because if the Pod is chosen as leader
    // in the meantime, it will be removed from nonLeadersToBeNotified so the notification will not be sent.
    scheduleNotLeaderNotification(event.getPod().getMetadata().getUid());

    if (leaderUID == null) {
      scheduleChooseNewLeader();
    }
  }

  public void onAgentPodDeleted(@ObservesAsync AgentPodDeleted event) {
    Pod pod = agentPods.remove(event.getUid());
    if (pod == null) {
      return;
    }
    LOGGER.info("Agent Pod " + name(pod) + " has been deleted.");
    if (event.getUid().equals(leaderUID)) {
      LOGGER.info("The leading agent Pod " + name(pod) + " has been removed. The operator will try to choose a new" +
              " agent Pod to monitor the Kubernetes cluster.");
      leaderUID = null;
      customResourceState.clearLeadingAgentPod();
      // If we do this right away we will choose a new leader while the daemon set is shutting down.
      // Give it a few seconds slack so that agentPods becomes empty when the daemon set is removed.
      scheduleChooseNewLeader();
    }
  }

  private void chooseNewLeader() {
    if (leaderUID != null) {
      return; // this might happen if we receive multiple AgentPodAdded events and each triggers chooseNewLeader.
    }
    if (agentPods.isEmpty()) {
      LOGGER.info("No running agent Pod available. Waiting for agent Pods to come up...");
      return;
    }
    String uid;
    Optional<String> existingUid = customResourceState.getLeadingAgentUid();
    if (existingUid.isPresent() && agentPods.containsKey(existingUid.get())) {
      uid = existingUid.get();
    } else {
      uid = randomKey(agentPods);
    }
    Pod pod = agentPods.get(uid);
    LOGGER.debug("Trying to choose agent Pod " + name(pod) + " as the leader.");
    if (setLeaderStatus(pod, true)) {
      leaderUID = uid;
      String msg = "Agent Pod " + name(pod) + " successfully became leader. It will monitor the Kubernetes cluster.";
      LOGGER.info(msg);
      kubernetesEventService.createKubernetesEvent(pod.getMetadata().getNamespace(), "AgentLeaderElected", msg, pod);
      customResourceState.updateLeadingAgentPod(pod);
    } else {
      LOGGER.debug("Could not choose agent Pod " + name(pod) + " as the leader.");
      scheduleChooseNewLeader();
    }
  }

  private void scheduleChooseNewLeader() {
    executor.schedule(this::chooseNewLeader, 5, TimeUnit.SECONDS);
  }

  private boolean setLeaderStatus(Pod pod, boolean isLeader) {
    if (notifyPod(pod, isLeader)) {
      // Success.
      nonLeadersToBeNotified.remove(pod.getMetadata().getUid());
      return true;
    } else {
      // Failed.
      // In case the error was a timeout, we don't know if the call was successful or not.
      // Schedule a notification to tell the Pod it's not the leader.
      scheduleNotLeaderNotification(pod.getMetadata().getUid());
      return false;
    }
  }

  private boolean notifyPod(Pod pod, boolean isLeader) {
    try {
      MediaType contentType = MediaType.get("text/yaml");
      String content = "com.instana.plugin.kubernetes.leader: " + isLeader;
      String ip = pod.getStatus().getHostIP();
      Request request = new Request.Builder()
          .url("http://" + ip + ":" + AGENT_PORT + "/config/com.instana.plugin.kubernetes")
          .post(RequestBody.create(contentType, content))
          .build();
      try (Response response = httpClient.newCall(request).execute()) {
        if (!response.isSuccessful()) {
          throw new IOException("Agent Pod responded with HTTP " + response.code() + errMsg(response));
        }
        LOGGER.debug("Successfully notified Pod " + name(pod) + " that it is" + (isLeader ? "" : " not") +
            " the leader.");
        return true; // success
      }
    } catch (Exception e) {
      // Log in DEBUG level because connection errors are expected while Pods are starting up.
      LOGGER.debug("Could not notify Pod " + name(pod) + " that it is" + (isLeader ? "" : " not") + " the leader: " +
          e.getMessage() + ".");
    }
    return false; // failed
  }

  private void scheduleNotLeaderNotification(String uid) {
    boolean wasFirst = nonLeadersToBeNotified.isEmpty();
    nonLeadersToBeNotified.add(uid);
    if (wasFirst) {
      executor.schedule(this::processPendingNotifications, 5, TimeUnit.SECONDS);
    }
  }

  private void processPendingNotifications() {
    if (nonLeadersToBeNotified.isEmpty()) {
      return;
    }
    String[] uids = nonLeadersToBeNotified.toArray(new String[]{});
    for (String uid : uids) {
      Pod pod = agentPods.get(uid);
      if (pod == null) {
        nonLeadersToBeNotified.remove(uid);
      } else {
        if (setLeaderStatus(pod, false)) {
          nonLeadersToBeNotified.remove(uid);
        }
      }
    }
    if (!nonLeadersToBeNotified.isEmpty()) {
      executor.schedule(this::processPendingNotifications, 5, TimeUnit.SECONDS);
    }
  }

  private String errMsg(Response response) {
    try {
      return ": " + response.body().string();
    } catch (Exception e) {
      return "";
    }
  }

  private String randomKey(Map<String, ?> map) {
    String[] keys = map.keySet().toArray(new String[]{});
    return keys[random.nextInt(keys.length)];
  }
}
