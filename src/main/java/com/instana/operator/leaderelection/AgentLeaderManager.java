package com.instana.operator.leaderelection;

import java.util.List;
import java.util.Optional;
import java.util.concurrent.ThreadLocalRandom;
import java.util.concurrent.atomic.AtomicReference;
import java.util.function.Predicate;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.event.Event;
import javax.enterprise.event.Observes;
import javax.inject.Inject;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.SerializationFeature;
import com.instana.operator.GlobalErrorEvent;
import com.instana.operator.agent.AgentConfigRestClient;
import com.instana.operator.customresource.ElectedLeaderSpec;
import com.instana.operator.service.ElectedLeaderClientService;
import com.instana.operator.service.InstanaConfigService;
import com.instana.operator.service.KubernetesResourceService;
import com.instana.operator.service.OperatorNamespaceService;
import com.instana.operator.service.ResourceCache;

import io.fabric8.kubernetes.api.model.Pod;
import io.quarkus.runtime.ShutdownEvent;
import io.reactivex.disposables.Disposable;

@ApplicationScoped
public class AgentLeaderManager {

  private static final Logger LOGGER = LoggerFactory.getLogger(AgentLeaderManager.class);

  // This is not typically configurable, so don't need to expose it to configuration here either.
  private static final int AGENT_PORT = 42699;
  private static final ObjectMapper MAPPER = new ObjectMapper();

  static {
    MAPPER.configure(SerializationFeature.INDENT_OUTPUT, true);
  }

  @Inject
  KubernetesResourceService clientService;
  @Inject
  OperatorNamespaceService namespaceService;
  @Inject
  InstanaConfigService instanaConfigService;
  @Inject
  AgentConfigRestClient agentConfigRestClient;
  @Inject
  ElectedLeaderClientService electedLeaderClientService;
  @Inject
  Event<GlobalErrorEvent> globalErrorEvent;

  private final Predicate<Pod> belongsToInstanaAgentDaemonSet = pod -> Optional.ofNullable(pod)
      .map(Pod::getMetadata)
      .flatMap(m -> Optional.ofNullable(m.getOwnerReferences()))
      .map(refs -> refs.stream().anyMatch(ownerRef ->
          "DaemonSet".equals(ownerRef.getKind()) && instanaConfigService.getConfig().getDaemonSetName()
              .equals(ownerRef.getName())))
      .orElse(false);

  private final Predicate<Pod> isRunning = pod -> Optional.ofNullable(pod)
      .map(Pod::getStatus)
      .filter(st -> !"Pending".equals(st.getPhase()))
      .flatMap(s -> Optional.ofNullable(s.getContainerStatuses()))
      .map(s -> s.stream().noneMatch(st -> null == st.getState().getRunning()))
      .orElse(false);

  private final AtomicReference<String> leaderName = new AtomicReference<>();

  private ResourceCache<Pod> agentPods;
  private Disposable watch;

  void onLeaderElection(@Observes LeaderElectionEvent ev) {
    if (!ev.isLeader()) {
      LOGGER.debug("Not the leader, so not doing anything.");
      if (null != watch) {
        watch.dispose();
      }
      return;
    }

    agentPods = clientService.createResourceCache("agents", client ->
        client.pods()
            .inNamespace(namespaceService.getNamespace())
            .withLabel("agent.instana.io/role", "agent"));

    watch = agentPods.observe()
        .subscribe(changeEvent -> {
          boolean running = isRunning.test(changeEvent.getNextValue());
          if (!running
              && changeEvent.getName().equals(leaderName.get())) {
            if (maybeStopBecomingLeader(changeEvent.getName())) {
              maybeNominateLeader();
            }
            return;
          }

          if (!belongsToInstanaAgentDaemonSet.test(changeEvent.getNextValue())
              && changeEvent.getName().equals(leaderName.get())) {
            if (maybeStopBecomingLeader(changeEvent.getName())) {
              maybeNominateLeader();
            }
            return;
          }

          if (changeEvent.isDeleted()) {
            if (maybeStopBecomingLeader(changeEvent.getName())) {
              maybeNominateLeader();
            }
            return;
          }

          maybeNominateLeader();
        }, ex -> LOGGER.error("Encountered an error in the AgentLeaderManager watch: " + ex.getMessage(), ex));
  }

  void onShutdown(@Observes ShutdownEvent _ev) {
    if (null != watch) {
      watch.dispose();
    }
  }

  private boolean isLeader(Pod p) {
    return p.getMetadata().getName().equals(leaderName.get());
  }

  private boolean maybeStopBecomingLeader(String name) {
    boolean wasLeader = leaderName.compareAndSet(name, null);
    if (wasLeader) {
      LOGGER.debug("Pod {} was previously leader. Removed.", name);
    }
    return wasLeader;
  }

  private void maybeNominateLeader() {
    // Leader has already been selected. Maybe.
    if (null != leaderName.get()) {
      boolean running = agentPods.get(leaderName.get())
          .map(isRunning::test)
          .orElse(false);
      if (running) {
        LOGGER.debug("Running leader {} has already been nominated. Skipping.", leaderName.get());
        return;
      } else {
        leaderName.set(null);
      }
    }

    // Maybe set current leader based on pre-elected leader (we're taking over for a failed Operator).
    LOGGER.debug("Checking whether a previously-elected leader was referenced...");
    electedLeaderClientService.loadElectedLeader()
        .map(ElectedLeaderSpec::getLeaderName)
        .flatMap(l -> agentPods.get(l))
        .filter(isRunning::test)
        .ifPresent(p -> {
          LOGGER.debug("Found previously-elected leader: {}", p.getMetadata().getName());
          leaderName.set(p.getMetadata().getName());
        });
    if (null != leaderName.get()) {
      // TODO: de-duplicate with ^^^
      boolean running = agentPods.get(leaderName.get())
          .map(isRunning::test)
          .orElse(false);
      if (running) {
        LOGGER.debug("Leader already elected {}.", leaderName.get());
        return;
      } else {
        leaderName.set(null);
      }
    }

    // Get current agent Pods as list.
    List<Pod> podsAsList = agentPods.toList();
    LOGGER.debug("Choosing a random leader from {} possibilities", podsAsList.size());
    Pod nominatedLeader = null;
    for (int tries = 0; tries < 3; tries++) {
      // Choose a leader at random.
      int leaderIdx = ThreadLocalRandom.current().nextInt(0, podsAsList.size());
      nominatedLeader = podsAsList.get(leaderIdx);
      if (isRunning.test(nominatedLeader)) {
        break;
      } else {
        nominatedLeader = null;
      }
    }

    if (null == nominatedLeader) {
      LOGGER.error("Couldn't find a running Pod to nominate as leader");
      return;
    }

    if (!leaderName.compareAndSet(null, nominatedLeader.getMetadata().getName())) {
      LOGGER.warn("Tried to elect Pod {} as leader, but {} was already elected.",
          nominatedLeader.getMetadata().getName(),
          leaderName.get());
      return;
    } else {
      LOGGER.debug("Firing agent leader config update...");
      fireAgentLeaderElectedEvent(nominatedLeader);
    }

    // Update the CRD for the current leader.
    LOGGER.debug("Updating ElectedLeader with leader name.");
    electedLeaderClientService.upsertElectedLeader(new ElectedLeaderSpec(leaderName.get()));
  }

  private void fireAgentLeaderElectedEvent(Pod leader) {
    LOGGER.debug("{} is becoming leader.", leader.getMetadata().getName());
    notifyAgent(leader, isLeader(leader));

    clientService.sendEvent(
        "agent-leader-elected",
        leader.getMetadata().getNamespace(),
        "ElectedLeader",
        "Successfully elected leader: " + leader.getMetadata().getNamespace() + "/" + leader.getMetadata().getName(),
        leader.getApiVersion(),
        leader.getKind(),
        leader.getMetadata().getNamespace(),
        leader.getMetadata().getName(),
        leader.getMetadata().getUid());
  }

  private void notifyAgent(Pod p, boolean isLeader) {
    agentConfigRestClient
        .updateAgentLeaderStatus(p.getStatus().getHostIP(), AGENT_PORT, isLeader)
        .blockingGet();
  }

}

