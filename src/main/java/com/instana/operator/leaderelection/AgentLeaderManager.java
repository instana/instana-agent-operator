package com.instana.operator.leaderelection;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.SerializationFeature;
import com.instana.operator.GlobalErrorEvent;
import com.instana.operator.agent.AgentConfigRestClient;
import com.instana.operator.customresource.ElectedLeaderSpec;
import com.instana.operator.service.*;
import io.fabric8.kubernetes.api.model.Pod;
import io.quarkus.runtime.ShutdownEvent;
import io.reactivex.disposables.Disposable;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.event.Event;
import javax.enterprise.event.Observes;
import javax.inject.Inject;
import java.util.List;
import java.util.Objects;
import java.util.Optional;
import java.util.concurrent.ThreadLocalRandom;
import java.util.concurrent.atomic.AtomicReference;
import java.util.function.Predicate;

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
          "DaemonSet".equals(ownerRef.getKind()) && instanaConfigService.getConfig().getDaemonSetName().equals(ownerRef.getName())))
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

    agentPods = clientService.createResourceCache(client ->
        client.pods()
            .inNamespace(namespaceService.getNamespace())
            .withLabel("agent.instana.io/role", "agent"));

    watch = agentPods.observe()
        .filter(changeEvent -> {
          if (!belongsToInstanaAgentDaemonSet.test(changeEvent.getNextValue())) {
            LOGGER.debug("Ignoring Pod {}, which doesn't belong to the agent DaemonSet.", changeEvent.getName());
            return false;
          } else {
            return isRunning.test(changeEvent.getNextValue());
          }
        })
        .subscribe(changeEvent -> {
          if (changeEvent.isDeleted()) {
            if (leaderName.compareAndSet(changeEvent.getName(), null)) {
              LOGGER.debug("DELETED Pod {} was the leader. Removed it and will nominate a new one.",
                  changeEvent.getName());
            }
          } else if (changeEvent.isAdded() && isRunning.test(changeEvent.getNextValue())) {
            // TODO: Can we remove this?
            // The custom resource is only there if another operator instance had created it and
            // this operator instance became leader after that. However, in that case we get initial ADDED
            // events for all agents in random order, and the algorithm below only works if the first ADDED event
            // happens to be for the current leader (otherwise we nominate the first agent where we get the ADDED event).
            // I think it would be easier if maybeNominateLeader() was checking the custom resource before choosing
            // the leader randomly, then we just call maybeNominateLeader() in all other places.
            LOGGER.debug("ADDED running Pod {}", changeEvent.getName());
            boolean isPodLeader = electedLeaderClientService.loadElectedLeader()
                .map(el -> changeEvent.getName().equals(el.getLeaderName()))
                .orElse(false);
            if (isPodLeader) {
              LOGGER.debug("Pod {} is the leader, so setting it.", changeEvent.getName());
              // This is the leader so short-circuit the nomination process.
              leaderName.set(changeEvent.getName());
              fireAgentLeaderElectedEvent(changeEvent.getNextValue());
              return;
            }
          } else if (null == leaderName.get()) {
            // TODO: This is also called if the event is ADDED and the pos it not running.
            LOGGER.debug("MODIFIED Pod {} and no leader nominated.", changeEvent.getName());
            // MODIFIED
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

  private void maybeNominateLeader() {
    // Leader has already been selected.
    if (null != leaderName.get()) {
      LOGGER.debug("Leader {} has already been nominated. Skipping.", leaderName.get());
      return;
    }

    // Get current agent Pods as list, choose one, try to make it the leader.
    List<Pod> podsAsList = agentPods.toList();
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
      globalErrorEvent
          .fire(new GlobalErrorEvent(new IllegalStateException("Couldn't find a running Pod to nominate as leader")));
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

