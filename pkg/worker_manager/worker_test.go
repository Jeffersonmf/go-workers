package workermanager

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestWorker_RunsDispatcherThenNestedCallback(t *testing.T) {
	t.Parallel()

	var dispatched, callbackReceived atomic.Bool

	Worker{
		SourceContext: context.Background(),
		Instrumentation: Instrumentation{
			TaskArguments: NewTaskParams(),
			FuncDispatcher: func(_ context.Context, _ TaskParams) (TaskParams, error) {
				dispatched.Store(true)
				out := NewTaskParams()
				SetParam(out, "ok", true)
				return out, nil
			},
			NestedCallback: func(_ context.Context, args TaskParams) error {
				ok, _ := GetParam[bool](args, "ok")
				callbackReceived.Store(ok)
				return nil
			},
			FuncName: "test-task",
		},
	}.Run()

	if !dispatched.Load() {
		t.Fatal("FuncDispatcher was never called")
	}
	if !callbackReceived.Load() {
		t.Fatal("NestedCallback did not receive the dispatcher's result")
	}
}

func TestWorker_RetriesUntilSuccessWithoutRacing(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32

	// Run under `go test -race`: the previous implementation signaled
	// retry via a buffered channel sent from inside a goroutine, then
	// immediately checked it with a non-blocking select in the caller
	// — a data race between the send and the check, and one that also
	// meant the retry almost never actually fired (the caller's select
	// usually reached its default case before the goroutine had run at
	// all). dispatchWithRetry runs entirely in the calling goroutine
	// instead, so there is nothing to race.
	Worker{
		SourceContext: context.Background(),
		Instrumentation: Instrumentation{
			TaskArguments: NewTaskParams(),
			FuncDispatcher: func(_ context.Context, _ TaskParams) (TaskParams, error) {
				n := attempts.Add(1)
				if n < 3 {
					return TaskParams{}, errors.New("not yet")
				}
				return NewTaskParams(), nil
			},
			FuncName: "flaky-task",
		},
	}.Run()

	if got := attempts.Load(); got != 3 {
		t.Fatalf("attempts = %d; want exactly 3 (2 failures + 1 success)", got)
	}
}

func TestWorker_ReportsErrorAfterExhaustingRetries(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32
	var reported atomic.Pointer[TaskError]

	Worker{
		SourceContext: context.Background(),
		MaxRetries:    2,
		Instrumentation: Instrumentation{
			TaskArguments: NewTaskParams(),
			FuncDispatcher: func(_ context.Context, _ TaskParams) (TaskParams, error) {
				attempts.Add(1)
				return TaskParams{}, errors.New("always fails")
			},
			NestedCallback: func(_ context.Context, _ TaskParams) error {
				t.Error("NestedCallback must not run when the dispatcher never succeeds")
				return nil
			},
			FuncName: "doomed-task",
		},
		OnError: func(err *TaskError) {
			reported.Store(err)
		},
	}.Run()

	if got := attempts.Load(); got != 2 {
		t.Fatalf("attempts = %d; want exactly MaxRetries (2)", got)
	}

	err := reported.Load()
	if err == nil {
		t.Fatal("OnError was never called")
	}
	if err.TaskName != "doomed-task" || err.Attempt != 2 {
		t.Fatalf("reported error = %+v; want TaskName=doomed-task Attempt=2", err)
	}
}

func TestWorker_RecoversFromAPanicInsteadOfCrashing(t *testing.T) {
	t.Parallel()

	var reported atomic.Pointer[TaskError]

	// This is exactly the scenario ListenErrosHappned was meant to
	// guard against in the previous implementation, but nothing ever
	// called it: a task panicking took the whole process down with it.
	Worker{
		SourceContext: context.Background(),
		MaxRetries:    1,
		Instrumentation: Instrumentation{
			TaskArguments: NewTaskParams(),
			FuncDispatcher: func(_ context.Context, _ TaskParams) (TaskParams, error) {
				panic("boom")
			},
			FuncName: "panicking-task",
		},
		OnError: func(err *TaskError) {
			reported.Store(err)
		},
	}.Run()

	err := reported.Load()
	if err == nil {
		t.Fatal("OnError was never called; the panic was not recovered")
	}
	if !strings.Contains(err.Err.Error(), "boom") {
		t.Fatalf("recovered error = %q; want it to mention the panic value \"boom\"", err.Err.Error())
	}
}

func TestWorker_ReportsNestedCallbackErrors(t *testing.T) {
	t.Parallel()

	var reported atomic.Pointer[TaskError]

	Worker{
		SourceContext: context.Background(),
		Instrumentation: Instrumentation{
			TaskArguments: NewTaskParams(),
			FuncDispatcher: func(_ context.Context, _ TaskParams) (TaskParams, error) {
				return NewTaskParams(), nil
			},
			NestedCallback: func(_ context.Context, _ TaskParams) error {
				return errors.New("downstream failed")
			},
			FuncName: "chained-task",
		},
		OnError: func(err *TaskError) {
			reported.Store(err)
		},
	}.Run()

	err := reported.Load()
	if err == nil {
		t.Fatal("OnError was never called for the failing NestedCallback")
	}
	if !strings.Contains(err.TaskName, "nested callback") {
		t.Fatalf("reported TaskName = %q; want it to identify the nested callback", err.TaskName)
	}
}

func TestWorker_RunWaitsForEveryConcurrentExecutionInATick(t *testing.T) {
	t.Parallel()

	var completed atomic.Int32
	const execsInTick = 20

	// The previous implementation launched these goroutines and
	// returned without waiting: Run() could report "done" while task
	// goroutines were still running. If that were still true here,
	// `completed` would almost never read execsInTick immediately after
	// Run() returns.
	Worker{
		SourceContext: context.Background(),
		ExecsPerTick:  []int{execsInTick},
		TickDuration:  time.Millisecond,
		Instrumentation: Instrumentation{
			TaskArguments: NewTaskParams(),
			FuncDispatcher: func(_ context.Context, _ TaskParams) (TaskParams, error) {
				time.Sleep(time.Millisecond)
				completed.Add(1)
				return NewTaskParams(), nil
			},
			FuncName: "concurrent-task",
		},
	}.Run()

	if got := completed.Load(); got != execsInTick {
		t.Fatalf("completed = %d immediately after Run() returned; want %d (Run must wait for every execution)", got, execsInTick)
	}
}

func TestWorker_StopsBetweenTicksWhenContextIsCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	var executions atomic.Int32

	done := make(chan struct{})
	go func() {
		defer close(done)
		Worker{
			SourceContext: ctx,
			ExecsPerTick:  []int{1, 1, 1, 1, 1},
			TickDuration:  50 * time.Millisecond,
			Instrumentation: Instrumentation{
				TaskArguments: NewTaskParams(),
				FuncDispatcher: func(_ context.Context, _ TaskParams) (TaskParams, error) {
					executions.Add(1)
					return NewTaskParams(), nil
				},
				FuncName: "cancellable-task",
			},
		}.Run()
	}()

	time.Sleep(70 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run() did not return promptly after context cancellation")
	}

	if got := executions.Load(); got >= 5 {
		t.Fatalf("executions = %d; want fewer than all 5 ticks, cancellation should have cut it short", got)
	}
}
