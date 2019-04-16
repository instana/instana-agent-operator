package com.instana.operator.leaderelection;

public class LeaderElectionEvent {

  private final boolean leader;

  public LeaderElectionEvent(boolean leader) {
    this.leader = leader;
  }

  public boolean isLeader() {
    return leader;
  }

  @Override
  public String toString() {
    return "LeaderElectionEvent{" +
        "leader=" + leader +
        '}';
  }

}
