package com.instana.operator.customresource;

import io.fabric8.kubernetes.api.builder.Function;
import io.fabric8.kubernetes.client.CustomResourceDoneable;

public class DoneableInstanaAgent extends CustomResourceDoneable<InstanaAgent> {
  public DoneableInstanaAgent(InstanaAgent resource, Function function) {
    super(resource, function);
  }
}
