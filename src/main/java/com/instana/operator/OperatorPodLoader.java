package com.instana.operator;

import com.instana.operator.cache.ResourceWatch;
import com.instana.operator.cache.ResourceService;
import com.instana.operator.events.OperatorPodRunning;
import io.fabric8.kubernetes.api.model.Pod;
import io.fabric8.kubernetes.api.model.PodList;
import io.fabric8.kubernetes.client.DefaultKubernetesClient;
import io.quarkus.runtime.StartupEvent;
import io.reactivex.disposables.Disposable;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.event.Event;
import javax.enterprise.event.NotificationOptions;
import javax.enterprise.event.Observes;
import javax.inject.Inject;
import javax.inject.Named;
import java.util.Optional;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.TimeoutException;

import static com.instana.operator.env.NamespaceProducer.OPERATOR_NAMESPACE;
import static com.instana.operator.env.OperatorPodNameProducer.POD_NAME;
import static com.instana.operator.resource.KubernetesResource.RUNNING;
import static com.instana.operator.resource.KubernetesResource.isRunning;

@ApplicationScoped
public class OperatorPodLoader {

  private static final Logger LOGGER = LoggerFactory.getLogger(OperatorPodLoader.class);

  @Inject
  DefaultKubernetesClient client;
  @Inject
  ResourceService resourceService;
  @Inject
  @Named(OPERATOR_NAMESPACE)
  String operatorNamespace;
  @Inject
  @Named(POD_NAME)
  String podName;
  @Inject
  FatalErrorHandler fatalErrorHandler;
  @Inject
  NotificationOptions asyncSerial;
  @Inject
  Event<OperatorPodRunning> operatorPodRunning;

  public void onStartup(@Observes StartupEvent _ev) {
    asyncSerial.getExecutor().execute(() -> {
      Pod myself = loadMyself(2, TimeUnit.MINUTES);
      LOGGER.debug("Pod " + operatorNamespace + "/" + podName + " is " + RUNNING + ".");
      operatorPodRunning.fireAsync(new OperatorPodRunning(myself), asyncSerial)
          .exceptionally(fatalErrorHandler::logAndExit);
    });
  }

  private Pod loadMyself(int timeout, TimeUnit unit) {
    CompletableFuture<Pod> myself = new CompletableFuture<>();
    ResourceWatch<Pod, PodList> podResourceWatch = resourceService.newResourceWatch(PodList.class);
    Disposable watch = podResourceWatch.listThenWatch(client.inNamespace(operatorNamespace).pods()).subscribe(event -> {
      Optional<Pod> pod = podResourceWatch.get(event.getUid()).filter(p -> podName.equals(p.getMetadata().getName()));
      if (pod.isPresent()) {
        if (isRunning(pod.get())) {
          myself.complete(pod.get());
        } else {
          String phase = pod.get().getStatus().getPhase();
          LOGGER.info("Pod " + operatorNamespace + "/" + podName + " is in phase " + phase.toLowerCase() + "." +
              " Waiting until it's " + RUNNING + ".");
        }
      }
    });
    try {
      return myself.get(timeout, unit);
    } catch (TimeoutException e) {
      LOGGER.error("Initialization error: Timeout while waiting for Pod " + operatorNamespace + "/" + podName +
          " to enter phase " + RUNNING + ".");
      fatalErrorHandler.systemExit(-1);
    } catch (Exception e) {
      LOGGER.error("Error while waiting for Pod " + operatorNamespace + "/" + podName + ": " + e.getMessage(), e);
      fatalErrorHandler.systemExit(-1);
    } finally {
      watch.dispose();
    }
    return null; // will not happen, because we call System.exit() in the catch clauses.
  }
}
