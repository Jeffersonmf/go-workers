package workermanager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Jeffersonmf/go-workers/pkg/util"
)

// defaultMaxRetries is used when Worker.MaxRetries is left at zero.
const defaultMaxRetries = 3

// Worker runs an Instrumentation's FuncDispatcher, optionally on a
// schedule (CronSchedulerConfig) and optionally as repeated ticks of
// concurrent executions (ExecsPerTick/TickDuration).
type Worker struct {
	SourceContext       context.Context
	Instrumentation     Instrumentation
	Callback            func(ctx context.Context)
	ExecsPerTick        []int
	TickDuration        time.Duration
	CronSchedulerConfig CronSchedulerConfig

	// MaxRetries caps how many times a failing FuncDispatcher call is
	// retried before it is reported via OnError. Zero uses
	// defaultMaxRetries.
	MaxRetries int

	// OnError is called for every task failure that exhausts its
	// retries, and for every NestedCallback failure. A nil OnError logs
	// through util.Logger instead. This replaced a global package-level
	// error-handling struct (WorkerError) that every Worker in the
	// process shared and mutated, which meant one worker's error state
	// could be overwritten by another's before anything read it.
	OnError func(err *TaskError)
}

// Run executes the worker: on its configured schedule if
// CronSchedulerConfig.IntervalInSeconds is set, otherwise once,
// immediately, blocking until every execution (and retry) it starts
// has finished.
func (wr Worker) Run() {
	if wr.CronSchedulerConfig.IntervalInSeconds > 0 {
		interval := time.Duration(wr.CronSchedulerConfig.IntervalInSeconds) * time.Second
		runOnSchedule(wr.context(), interval, wr.CronSchedulerConfig.TypeOfExecution, wr.blockToRun)
		return
	}
	wr.blockToRun()
}

func (wr Worker) context() context.Context {
	if wr.SourceContext == nil {
		return context.Background()
	}
	return wr.SourceContext
}

func (wr Worker) blockToRun() {
	ctx := wr.context()

	if wr.TickDuration <= 0 || wr.ExecsPerTick == nil {
		start := time.Now()
		wr.executeTask(ctx, wr.Instrumentation.TaskArguments.Clone())
		util.Logger.Info("worker finished",
			"name", wr.Instrumentation.FuncName,
			"elapsed", time.Since(start).String(),
		)
	} else {
		wr.runTicks(ctx)
	}

	if wr.Callback != nil {
		wr.Callback(ctx)
	}
}

// runTicks runs each element of ExecsPerTick as one tick: that many
// concurrent executions of the task, waited for (sync.WaitGroup)
// before the next tick starts. The previous implementation launched
// these goroutines and returned immediately without waiting for them,
// so a caller's main() could exit — and, in a cron/container context,
// the process could be considered "done" — while task goroutines were
// still mid-flight. Waiting here makes one worker's ticks run in a
// predictable order and guarantees every execution it started has
// either finished or reported an error before blockToRun returns.
func (wr Worker) runTicks(ctx context.Context) {
	execs := wr.ExecsPerTick

	for tick, count := range execs {
		if ctx.Err() != nil {
			return
		}

		start := time.Now()

		var wg sync.WaitGroup
		for i := 0; i < count; i++ {
			taskArg := wr.Instrumentation.TaskArguments.Clone()
			SetParam(taskArg, "currentTick", tick)
			SetParam(taskArg, "ticksLeft", len(execs)-tick)
			SetParam(taskArg, "currentExec", i)

			wg.Add(1)
			go func(taskArg TaskParams) {
				defer wg.Done()
				wr.executeTask(ctx, taskArg)
			}(taskArg)
		}
		wg.Wait()

		elapsed := time.Since(start)
		if elapsed >= wr.TickDuration {
			util.Logger.Warn("tick took longer than the configured tick duration",
				"worker", wr.Instrumentation.FuncName,
				"tick", tick,
				"elapsed", elapsed.String(),
				"tickDuration", wr.TickDuration.String(),
			)
			continue
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(wr.TickDuration - elapsed):
		}
	}
}

// executeTask runs FuncDispatcher with retries and panic recovery,
// then, on success, its NestedCallback if one is set. Failures that
// exhaust their retries are reported via reportError instead of being
// returned, because executeTask always runs inside a goroutine that
// nothing else observes the return value of (see runTicks); an error
// that is only ever returned into a discarded goroutine result is
// effectively silent, so reporting happens here, at the one place that
// actually knows the failure occurred.
func (wr Worker) executeTask(ctx context.Context, taskArg TaskParams) {
	maxRetries := wr.MaxRetries
	if maxRetries <= 0 {
		maxRetries = defaultMaxRetries
	}

	result, attempt, err := wr.dispatchWithRetry(ctx, taskArg, maxRetries)
	if err != nil {
		wr.reportError(&TaskError{TaskName: wr.Instrumentation.FuncName, Attempt: attempt, Err: err})
		return
	}

	if wr.Instrumentation.NestedCallback == nil {
		return
	}
	if err := wr.Instrumentation.NestedCallback(ctx, result); err != nil {
		wr.reportError(&TaskError{TaskName: wr.Instrumentation.FuncName + " (nested callback)", Attempt: 1, Err: err})
	}
}

func (wr Worker) dispatchWithRetry(ctx context.Context, taskArg TaskParams, maxRetries int) (result TaskParams, attempt int, err error) {
	for attempt = 1; attempt <= maxRetries; attempt++ {
		if ctx.Err() != nil {
			return TaskParams{}, attempt, ctx.Err()
		}

		result, err = wr.safeDispatch(ctx, taskArg)
		if err == nil {
			return result, attempt, nil
		}

		util.Logger.Warn("task attempt failed",
			"worker", wr.Instrumentation.FuncName,
			"attempt", attempt,
			"maxAttempts", maxRetries,
			"error", err,
		)
	}
	return TaskParams{}, attempt - 1, err
}

// safeDispatch runs FuncDispatcher and converts a panic into an error
// instead of letting it crash the process. FuncDispatcher is a
// caller-supplied function run inside a goroutine this package
// spawns; a panic in one task must not take down every other worker
// running in the same process. The previous implementation had a
// panic-recovery method (ListenErrosHappned) but nothing in the
// codebase ever called it, so no task execution was actually
// protected.
func (wr Worker) safeDispatch(ctx context.Context, taskArg TaskParams) (result TaskParams, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return wr.Instrumentation.FuncDispatcher(ctx, taskArg)
}

func (wr Worker) reportError(taskErr *TaskError) {
	if wr.OnError != nil {
		wr.OnError(taskErr)
		return
	}
	util.Logger.Error("task failed",
		"task", taskErr.TaskName,
		"attempt", taskErr.Attempt,
		"error", taskErr.Err,
	)
}
