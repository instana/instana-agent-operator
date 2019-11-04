package com.instana.operator.coordination;


import com.fasterxml.jackson.annotation.JsonCreator;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.util.Set;

public class CoordinationRecord {
  private final Set<String> requested;
  private final Set<String> assigned;

  @JsonCreator
  public CoordinationRecord(@JsonProperty("requested") Set<String> requested,
                            @JsonProperty("assigned") Set<String> assigned) {
    this.requested = requested;
    this.assigned = assigned;
  }

  public Set<String> getRequested() {
    return requested;
  }

  public Set<String> getAssigned() {
    return assigned;
  }

  @Override
  public String toString() {
    return "CoordinationRecord{" +
        "requested=" + requested +
        ", assigned=" + assigned +
        '}';
  }
}
