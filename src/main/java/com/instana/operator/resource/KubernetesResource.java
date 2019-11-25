package com.instana.operator.resource;

import com.instana.operator.customresource.InstanaAgent;
import io.fabric8.kubernetes.api.model.HasMetadata;
import io.fabric8.kubernetes.api.model.OwnerReference;
import io.fabric8.kubernetes.api.model.Pod;

import java.util.function.Predicate;

public class KubernetesResource {

  public static final String RUNNING = "running";

  public static boolean hasOwner(HasMetadata resource, InstanaAgent owner) {
    return hasOwnerUid(resource, owner.getMetadata().getUid());
  }

  public static boolean isRunning(Pod pod) {
    return RUNNING.equalsIgnoreCase(pod.getStatus().getPhase());
  }

  public static Predicate<Pod> isRunning() {
    return KubernetesResource::isRunning;
  }

  public static <T extends HasMetadata> Predicate<T> hasOwner(HasMetadata owner) {
    return configMap -> hasOwnerUid(configMap, owner.getMetadata().getUid());
  }

  public static <T extends HasMetadata> Predicate<T> hasOwner(OwnerReference ownerReference) {
    return configMap -> hasOwnerUid(configMap, ownerReference.getUid());
  }

  private static <T extends HasMetadata> boolean hasOwnerUid(T resource, String uid) {
    if (resource.getMetadata().getOwnerReferences() != null) {
      return resource.getMetadata().getOwnerReferences().stream()
          .anyMatch(o -> o.getUid().equals(uid));
    } else {
      return false;
    }
  }

  public static boolean hasSameName(HasMetadata a, HasMetadata b) {
    return a.getMetadata().getName().equals(b.getMetadata().getName());
  }

  public static String name(HasMetadata resource) {
    return resource.getMetadata().getNamespace() + "/" + resource.getMetadata().getName();
  }
}
