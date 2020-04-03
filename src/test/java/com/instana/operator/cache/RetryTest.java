package com.instana.operator.cache;

import io.reactivex.disposables.Disposable;
import io.reactivex.disposables.Disposables;
import io.reactivex.schedulers.TestScheduler;
import org.junit.jupiter.api.Test;

import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.function.Consumer;

import static org.hamcrest.MatcherAssert.assertThat;
import static org.hamcrest.Matchers.hasSize;
import static org.hamcrest.Matchers.is;

class RetryTest {
  @Test
  void must_restart_operation_if_it_fails() {
    final List<Retry.TaskCallback> invocations = new ArrayList<>();
    final TestScheduler scheduler = new TestScheduler();

    new Retry(noopDisposable(invocations::add), scheduler).start();

    assertThat(invocations, hasSize(1));
    invocations.get(0).done(new Exception("fake exception"));
    scheduler.advanceTimeBy(1, TimeUnit.SECONDS);

    assertThat(invocations, hasSize(2));
  }

  @Test
  void must_restart_with_longer_delay_after_multiple_failures() {
    final List<Retry.TaskCallback> invocations = new ArrayList<>();
    final TestScheduler scheduler = new TestScheduler();

    new Retry(noopDisposable(invocations::add), scheduler).start();

    assertThat(invocations, hasSize(1));
    invocations.get(0).done(new Exception("fake exception"));
    assertThat(invocations, hasSize(1));
    scheduler.advanceTimeBy(1, TimeUnit.SECONDS);
    assertThat(invocations, hasSize(2));
    invocations.get(1).done(new Exception("another exception"));
    assertThat(invocations, hasSize(2));
    scheduler.advanceTimeBy(1, TimeUnit.SECONDS);
    assertThat(invocations, hasSize(2));
    scheduler.advanceTimeBy(1, TimeUnit.SECONDS);
    assertThat(invocations, hasSize(3));
  }

  @Test
  void must_stop_when_disposed() {
    final List<Retry.TaskCallback> invocations = new ArrayList<>();
    final TestScheduler scheduler = new TestScheduler();

    final Disposable disposable = new Retry(noopDisposable(invocations::add), scheduler).start();
    assertThat(invocations, hasSize(1));
    invocations.get(0).done(new Exception("fake exception"));

    disposable.dispose();
    scheduler.advanceTimeBy(1, TimeUnit.SECONDS);
    assertThat(invocations, hasSize(1));
  }

  @Test
  void must_stop_backoff_task_when_disposed() {
    final List<Retry.TaskCallback> invocations = new ArrayList<>();
    final TestScheduler scheduler = new TestScheduler();
    final AtomicBoolean taskDisposed = new AtomicBoolean();

    final Disposable disposable = new Retry(callback -> {
      invocations.add(callback);
      return Disposables.fromAction(() -> taskDisposed.set(true));
    }, scheduler).start();
    assertThat(invocations, hasSize(1));
    invocations.get(0).done(new Exception("fake exception"));

    disposable.dispose();
    scheduler.advanceTimeBy(1, TimeUnit.SECONDS);
    assertThat(invocations, hasSize(1));

    assertThat(taskDisposed.get(), is(true));
  }

  static Retry.RetryRunnable noopDisposable(Consumer<Retry.TaskCallback> task) {
    return callback -> {
      task.accept(callback);
      return Disposables.empty();
    };
  }
}
