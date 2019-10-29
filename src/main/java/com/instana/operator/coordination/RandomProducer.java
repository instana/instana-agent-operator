package com.instana.operator.coordination;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.inject.Produces;
import javax.inject.Singleton;
import java.util.Random;

@ApplicationScoped
public class RandomProducer {
  @Produces
  @Singleton
  Random random() {
    return new Random();
  }
}
