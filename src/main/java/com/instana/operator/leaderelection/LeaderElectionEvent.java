package com.instana.operator.leaderelection;

public class LeaderElectionEvent {

  private final boolean leader;

  public LeaderElectionEvent(boolean leader) {
    this.leader = leader;
  }

  public boolean isLeader() {
    return leader;
  }

}
