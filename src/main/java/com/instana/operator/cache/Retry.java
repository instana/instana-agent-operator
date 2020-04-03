package com.instana.operator.cache;

import io.reactivex.Scheduler;
import io.reactivex.disposables.Disposable;
import io.reactivex.disposables.Disposables;

import java.time.Duration;
import java.util.Optional;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicReference;

public class Retry {
  public static final Duration INITIAL_DELAY = Duration.ofSeconds(1);
  private final RetryRunnable task;
  private final Scheduler scheduler;
  private final AtomicReference<Duration> delay = new AtomicReference<>(INITIAL_DELAY);
  private final AtomicReference<Disposable> scheduledDisposableRef = new AtomicReference<>();
  private final AtomicReference<Disposable> taskDisposableRef = new AtomicReference<>();

  public Retry(RetryRunnable task, Scheduler scheduler) {
    this.task = task;
    this.scheduler = scheduler;
  }

  Disposable start() {
    new TaskRunnable().run();
    return Disposables.fromAction(() -> {
      Optional.ofNullable(scheduledDisposableRef.getAndSet(null)).ifPresent(Disposable::dispose);
      Optional.ofNullable(taskDisposableRef.getAndSet(null)).ifPresent(Disposable::dispose);
    });
  }

  @FunctionalInterface
  interface RetryRunnable {
    Disposable run(TaskCallback callback);
  }

  @FunctionalInterface
  interface TaskCallback {
    void done(Exception e);
  }

  private class TaskRunnable implements Runnable {
    @Override
    public void run() {
      taskDisposableRef.set(task.run(ex -> {
        if (ex != null) {
          scheduledDisposableRef.set(scheduler.scheduleDirect(new TaskRunnable(), delay.get().toMillis(), TimeUnit.MILLISECONDS));
          delay.updateAndGet(cur -> cur.multipliedBy(2));
        }
      }));
    }
  }
}
