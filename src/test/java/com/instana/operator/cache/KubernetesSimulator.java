package com.instana.operator.cache;

import io.fabric8.kubernetes.api.model.KubernetesResourceList;
import io.fabric8.kubernetes.api.model.ListMeta;
import io.fabric8.kubernetes.api.model.ObjectMeta;
import io.fabric8.kubernetes.api.model.Pod;
import io.fabric8.kubernetes.api.model.PodList;
import io.fabric8.kubernetes.client.KubernetesClientException;
import io.fabric8.kubernetes.client.Watch;
import io.fabric8.kubernetes.client.Watcher;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;

public class KubernetesSimulator extends ListerWatcher<Pod, PodList> implements AutoCloseable {

  KubernetesSimulator() {
    super(null);
  }

  private final Map<String, Pod> pods = new HashMap<>(); // uid -> Pod
  private final ExecutorService watcherThread = Executors.newSingleThreadExecutor();
  private volatile Watcher<Pod> watcher;

  void simulatePodAdded(String uid, String name, int resourceVersion) {
    Pod pod = makePod(name, resourceVersion, uid);
    pods.put(uid, pod);
    if (watcher != null) {
      Pod clone = copy(pod);
      watcherThread.execute(() -> watcher.eventReceived(Watcher.Action.ADDED, clone));
    }
  }

  void simulatePodModified(String uid, int resourceVersion) {
    Pod pod = copy(pods.get(uid));
    pod.getMetadata().setResourceVersion("" + resourceVersion);
    pods.put(uid, pod);
    if (watcher != null) {
      Pod clone = copy(pod);
      watcherThread.execute(() -> watcher.eventReceived(Watcher.Action.MODIFIED, clone));
    }
  }

  void simulatePodDeleted(String uid) {
    Pod pod = pods.remove(uid);
    if (watcher != null) {
      Pod clone = copy(pod);
      watcherThread.execute(() -> watcher.eventReceived(Watcher.Action.DELETED, clone));
    }
  }

  void simulateError() {
    if (watcher != null) {
      watcherThread.execute(() -> watcher.onClose(new KubernetesClientException("simulated simulateError")));
    }
  }

  @Override
  public KubernetesResourceList<Pod> list() {
    return new KubernetesResourceList<Pod>() {
      @Override
      public ListMeta getMetadata() {
        return null;
      }

      @Override
      public List<Pod> getItems() {
        return new ArrayList<>(pods.values());
      }
    };
  }

  @Override
  public Watch watch(Watcher<Pod> watcher) {
    this.watcher = watcher;
    return () -> {
    };
  }

  private Pod makePod(String name, int resourceVersion, String uid) {
    Pod pod = new Pod();
    pod.setMetadata(new ObjectMeta());
    pod.getMetadata().setName(name);
    pod.getMetadata().setResourceVersion("" + resourceVersion);
    pod.getMetadata().setUid(uid);
    return pod;
  }

  private Pod copy(Pod pod) {
    String name = pod.getMetadata().getName();
    int resourceVersion = Integer.parseInt(pod.getMetadata().getResourceVersion());
    String uid = pod.getMetadata().getUid();
    return makePod(name, resourceVersion, uid);
  }

  @Override
  public void close() throws Exception {
    watcherThread.shutdown();
    if (!watcherThread.awaitTermination(5, TimeUnit.SECONDS)) {
      throw new Exception("timeout while terminating the kubernetes simulator executor thread");
    }
  }
}
