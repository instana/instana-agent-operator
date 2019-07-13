package com.instana.operator;

import com.instana.operator.customresource.DoneableInstanaAgent;
import com.instana.operator.customresource.InstanaAgent;
import com.instana.operator.customresource.InstanaAgentList;
import com.instana.operator.events.CustomResourceAdded;
import com.instana.operator.events.CustomResourceDeleted;
import com.instana.operator.events.CustomResourceModified;
import com.instana.operator.events.CustomResourceOtherInstanceAdded;
import io.fabric8.kubernetes.client.dsl.MixedOperation;
import io.fabric8.kubernetes.client.dsl.Resource;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.event.ObservesAsync;
import javax.inject.Inject;

import static com.instana.operator.client.KubernetesClientProducer.CRD_NAME;
import static com.instana.operator.util.ResourceUtils.name;

/**
 * On custom resource events, we need to make sure to first update the
 * resource state, then inform the agent deployer. While this could be implemented
 * by having multiple CDI event handlers with @Priority, it's easier to have
 * this class coordinating it. Moreover, this class is a good central place
 * for logging custom resource related debug messages.
 */
@ApplicationScoped
public class CustomResourceEventHandler {

  @Inject
  FatalErrorHandler fatalErrorHandler;
  @Inject
  MixedOperation<InstanaAgent, InstanaAgentList, DoneableInstanaAgent, Resource<InstanaAgent, DoneableInstanaAgent>> client;
  @Inject
  AgentDeployer agentDeployer;
  @Inject
  CustomResourceState customResourceState;

  private static final Logger LOGGER = LoggerFactory.getLogger(CustomResourceEventHandler.class);

  /**
   * Note that "added" means the operator sees the custom resource for the first time.
   * This can mean that the user actually created a new resource, but it can also mean
   * that this operator took over as leader and found an existing custom resource that
   * was managed by the previous leader.
   */
  void customResourceAdded(@ObservesAsync CustomResourceAdded event) {
    LOGGER.debug("Custom resource " + CRD_NAME + " " + name(event.getInstanaAgentResource()) + " has been added.");
    customResourceState.customResourceAdded(event.getInstanaAgentResource());
    agentDeployer.customResourceAdded(event.getInstanaAgentResource());
  }

  void customResourceDeleted(@ObservesAsync CustomResourceDeleted event) {
    LOGGER.info("Custom resource " + CRD_NAME + " " + name(event.getInstanaAgentResource()) + " has been deleted." +
        " The agent and all associated resources will be removed.");
    agentDeployer.customResourceDeleted();
    customResourceState.customResourceDeleted();
  }

  void onOtherInstanceAdded(@ObservesAsync CustomResourceOtherInstanceAdded event) {
    LOGGER.info("Custom resource " + CRD_NAME + " " + name(event.getNewInstance()) + " has been added, but " +
        name(event.getCurrentInstance()) + " already exists. Only one Instana agent can be installed." +
        " Ignoring the new resource.");
  }

  void onModified(@ObservesAsync CustomResourceModified event) {
    // In that case, the CustomResourceState will log a message.
    customResourceState.customResourceModified(event.getNext());
  }
}
