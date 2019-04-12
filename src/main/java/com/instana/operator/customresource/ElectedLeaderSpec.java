package com.instana.operator.customresource;

import com.fasterxml.jackson.databind.JsonDeserializer;
import com.fasterxml.jackson.databind.annotation.JsonDeserialize;

@JsonDeserialize(
    using = JsonDeserializer.None.class
)
public class ElectedLeaderSpec {

  private String leaderName;

  public ElectedLeaderSpec() {
  }

  public ElectedLeaderSpec(String leaderName) {
    this.leaderName = leaderName;
  }

  public String getLeaderName() {
    return leaderName;
  }

  public void setLeaderName(String leaderName) {
    this.leaderName = leaderName;
  }

  @Override
  public String toString() {
    return "leaderName=" + leaderName;
  }

}
