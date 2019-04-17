package com.instana.operator.leaderelection;

import java.util.Optional;
import java.util.concurrent.atomic.AtomicReference;
import java.util.function.Predicate;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.event.Event;
import javax.enterprise.event.Observes;
import javax.enterprise.event.ObservesAsync;
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
public class AgentLeaderElector {

  private static final Logger LOGGER = LoggerFactory.getLogger(AgentLeaderElector.class);

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

  private final Predicate<Pod> isRunning = pod -> Optional.ofNullable(pod)
      .map(Pod::getStatus)
      .filter(st -> !"Pending".equals(st.getPhase()))
      .flatMap(s -> Optional.ofNullable(s.getContainerStatuses()))
      .map(s -> s.stream().noneMatch(st -> null == st.getState().getRunning()))
      .orElse(false);

  private final AtomicReference<String> leaderName = new AtomicReference<>();

  private ResourceCache<Pod> agentPods;
  private Disposable watch;

  void onElectedLeader(@ObservesAsync ElectedLeaderEvent ev) {
    agentPods = clientService.createResourceCache("agents", client ->
        client.pods()
            .inNamespace(namespaceService.getNamespace())
            .withLabel("agent.instana.io/role", "agent"));

    watch = agentPods.observe()
        .groupBy(changeEvent -> isLeader(changeEvent.getNextValue()))
        .subscribe(events -> {
          if (events.getKey()) {
            // These events pertain to the leader.
            events
                .filter(changeEvent -> changeEvent.isDeleted() || !isRunning.test(changeEvent.getNextValue()))
                .subscribe(changeEvent -> {
                  // Elected leader no longer running. Nominate a new one.
                  LOGGER.debug("Elected leader was {} but no longer running. Setting to null.", changeEvent.getName());
                  leaderName.set(null);
                  maybeNominateLeader();
                });
          } else {
            // These events might affect leadership.
            events
                .subscribe(changeEvent -> {
                  if (changeEvent.isAdded() || changeEvent.isModified()) {
                    maybeNominateLeader();
                  }
                });
          }
        }, ex -> LOGGER.error("Encountered an error in the AgentLeaderElector watch: " + ex.getMessage(), ex));
  }

  void onImpeachedLeader(@ObservesAsync ImpeachedLeaderEvent _ev) {
    if (null != watch) {
      watch.dispose();
    }
  }

  void onShutdown(@Observes ShutdownEvent _ev) {
    if (null != watch) {
      watch.dispose();
    }
  }

  private boolean isLeader(Pod p) {
    return null != p && p.getMetadata().getName().equals(leaderName.get());
  }

  private boolean isLeaderRunning() {
    return agentPods.get(leaderName.get()).map(isRunning::test).orElse(false);
  }

  private void maybeNominateLeader() {
    if (null != leaderName.get() && isLeaderRunning()) {
      LOGGER.debug("Trying to nominate leader but leader already elected and still running.");
      return;
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
    if (null != leaderName.get() && isLeaderRunning()) {
      LOGGER.debug("Trying to nominate leader but leader already loaded from custom resource and still running.");
      return;
    }

    // Get current agent Pods as list.
    LOGGER.debug("Choosing first running Pod...");
    Optional<Pod> nominee = agentPods.toList().stream()
        .filter(isRunning)
        .findFirst();
    if (!nominee.isPresent()) {
      LOGGER.warn("Couldn't find a running Pod to nominate as leader.");
      return;
    }

    if (!leaderName.compareAndSet(null, nominee.get().getMetadata().getName())) {
      LOGGER.warn("Tried to elect Pod {} as leader, but {} was already elected.",
          nominee.get().getMetadata().getName(),
          leaderName.get());
      return;
    }

    LOGGER.debug("Firing agent leader config update...");
    fireAgentLeaderElectedEvent(nominee.get());

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

