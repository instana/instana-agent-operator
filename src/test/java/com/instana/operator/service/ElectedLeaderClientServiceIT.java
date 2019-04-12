package com.instana.operator.service;

import javax.inject.Inject;

import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;

import io.quarkus.test.junit.QuarkusTest;

@QuarkusTest
class ElectedLeaderClientServiceIT {

  @Inject
  ElectedLeaderClientService electedLeaderClientService;

  @BeforeEach
  void setUp() {
  }

  @AfterEach
  void tearDown() {
  }

}
