package com.instana.operator.cache;

import io.fabric8.kubernetes.api.model.HasMetadata;

import java.util.Collections;
import java.util.HashMap;
import java.util.Map;
import java.util.Optional;

class ResourceMap<T extends HasMetadata> {

  private final Map<String, T> map = Collections.synchronizedMap(new HashMap<>());

  /**
   * @return true if the map was updated, false otherwise.
   */
  boolean putIfNewer(String uid, T resource) {
    synchronized(map) {
      if (map.containsKey(uid)) {
        int currentResourceVersion = Integer.parseInt(map.get(uid).getMetadata().getResourceVersion());
        int newResourceVersion = Integer.parseInt(resource.getMetadata().getResourceVersion());
        if (currentResourceVersion < newResourceVersion) {
          map.put(uid, resource);
          return true;
        } else {
          return false;
        }
      } else {
        map.put(uid, resource);
        return true;
      }
    }
  }

  public Optional<T> get(String uid) {
    return Optional.ofNullable(map.get(uid));
  }

  /**
   * @return true if the map was updated, false otherwise.
   */
  boolean remove(String uid) {
    return map.remove(uid) != null;
  }
}
