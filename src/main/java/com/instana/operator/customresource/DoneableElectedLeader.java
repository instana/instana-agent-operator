package com.instana.operator.customresource;

import io.fabric8.kubernetes.api.builder.Function;
import io.fabric8.kubernetes.client.CustomResourceDoneable;

public class DoneableElectedLeader extends CustomResourceDoneable<ElectedLeader> {
  public DoneableElectedLeader(ElectedLeader resource, Function function) {
    super(resource, function);
  }
}
