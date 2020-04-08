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

    assertThat("task must have been started the first time", invocations, hasSize(1));
    invocations.get(0).done(new Exception("fake exception"));
    scheduler.advanceTimeBy(1, TimeUnit.SECONDS);

    assertThat("task must have been restarted", invocations, hasSize(2));
  }

  @Test
  void must_restart_with_longer_delay_after_multiple_failures() {
    final List<Retry.TaskCallback> invocations = new ArrayList<>();
    final TestScheduler scheduler = new TestScheduler();

    new Retry(noopDisposable(invocations::add), scheduler).start();

    assertThat("task must have been started the first time", invocations, hasSize(1));
    invocations.get(0).done(new Exception("fake exception"));
    assertThat("task must not have been restarted immediately after exception", invocations, hasSize(1));
    scheduler.advanceTimeBy(1, TimeUnit.SECONDS);
    assertThat("task must have been restarted after 1 second", invocations, hasSize(2));
    invocations.get(1).done(new Exception("another exception"));
    assertThat("task must not have been restarted immediately after second exception", invocations, hasSize(2));
    scheduler.advanceTimeBy(1, TimeUnit.SECONDS);
    assertThat("task must not have been restarted after 1 second", invocations, hasSize(2));
    scheduler.advanceTimeBy(1, TimeUnit.SECONDS);
    assertThat("task must have been restarted after 2 seconds", invocations, hasSize(3));
  }

  @Test
  void must_stop_when_disposed() {
    final List<Retry.TaskCallback> invocations = new ArrayList<>();
    final TestScheduler scheduler = new TestScheduler();

    final Disposable disposable = new Retry(noopDisposable(invocations::add), scheduler).start();
    assertThat("task must have been started the first time", invocations, hasSize(1));
    invocations.get(0).done(new Exception("fake exception"));

    disposable.dispose();
    scheduler.advanceTimeBy(1, TimeUnit.SECONDS);
    assertThat("task must not have been restarted", invocations, hasSize(1));
  }

  @Test
  void must_reset_delay_after_five_minute_have_passed() {
    final List<Retry.TaskCallback> invocations = new ArrayList<>();
    final TestScheduler scheduler = new TestScheduler();
    new Retry(noopDisposable(invocations::add), scheduler).start();

    assertThat("task must have been started the first time", invocations, hasSize(1));
    invocations.get(0).done(new Exception("some exception"));
    scheduler.advanceTimeBy(1, TimeUnit.SECONDS);
    invocations.get(1).done(new Exception("another exception"));
    scheduler.advanceTimeBy(2, TimeUnit.SECONDS);
    assertThat("task must have been restarted after each exception", invocations, hasSize(3));
    scheduler.advanceTimeBy(5, TimeUnit.MINUTES);
    invocations.get(2).done(new Exception("some exception after a long time"));
    scheduler.advanceTimeBy(1, TimeUnit.SECONDS);
    assertThat("task must have been restarted after initial delay", invocations, hasSize(4));
  }

  @Test
  void must_not_reset_if_another_error_occurs() {
    final List<Retry.TaskCallback> invocations = new ArrayList<>();
    final TestScheduler scheduler = new TestScheduler();
    new Retry(noopDisposable(invocations::add), scheduler).start();

    assertThat("task should have been started the first time", invocations, hasSize(1));
    invocations.get(0).done(new Exception("some exception"));
    scheduler.advanceTimeBy(1, TimeUnit.SECONDS);
    assertThat("task should have been restarted with 1 second delay after first error", invocations, hasSize(2));
    scheduler.advanceTimeBy(2, TimeUnit.MINUTES);
    invocations.get(1).done(new Exception("another exception"));
    scheduler.advanceTimeBy(2, TimeUnit.SECONDS);
    assertThat("task should have been restarted with 2 second delay after second error", invocations, hasSize(3));
    scheduler.advanceTimeBy(3, TimeUnit.MINUTES);
    invocations.get(2).done(new Exception("another exception"));
    scheduler.advanceTimeBy(1, TimeUnit.SECONDS);
    assertThat("task must not have been restarted after 1 second", invocations, hasSize(3));
    scheduler.advanceTimeBy(3, TimeUnit.SECONDS);
    assertThat("task must have been restarted after 4 seconds", invocations, hasSize(4));
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
    assertThat("task must have been started the first time", invocations, hasSize(1));
    invocations.get(0).done(new Exception("fake exception"));

    disposable.dispose();
    scheduler.advanceTimeBy(1, TimeUnit.SECONDS);
    assertThat("task must not be restarted after being disposed", invocations, hasSize(1));

    assertThat("task must be disposed", taskDisposed.get(), is(true));
  }

  static Retry.RetryRunnable noopDisposable(Consumer<Retry.TaskCallback> task) {
    return callback -> {
      task.accept(callback);
      return Disposables.empty();
    };
  }
}
