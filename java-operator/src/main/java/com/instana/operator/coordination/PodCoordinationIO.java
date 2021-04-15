/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc.
 */
package com.instana.operator.coordination;

import io.fabric8.kubernetes.api.model.Pod;

import java.io.IOException;
import java.util.Set;

public interface PodCoordinationIO {
  void assign(Pod pod, Set<String> assignment) throws IOException;

  CoordinationRecord pollPod(Pod pod) throws IOException;
}
