package workermanager

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunOnSchedule_RunsImmediatelyThenOnEachTick(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var calls atomic.Int32
	go runOnSchedule(ctx, 10*time.Millisecond, ExecutionAsync, func() {
		calls.Add(1)
	})

	// Immediate call, then at least two ticks: budget generously to
	// keep this from flaking on a loaded CI runner.
	deadline := time.After(500 * time.Millisecond)
	for calls.Load() < 3 {
		select {
		case <-deadline:
			t.Fatalf("calls = %d after 500ms; want at least 3 (1 immediate + ticks)", calls.Load())
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func TestRunOnSchedule_StopsOnContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	var calls atomic.Int32

	go runOnSchedule(ctx, 5*time.Millisecond, ExecutionAsync, func() {
		calls.Add(1)
	})

	time.Sleep(30 * time.Millisecond)
	cancel()
	time.Sleep(20 * time.Millisecond)
	countAtCancel := calls.Load()

	time.Sleep(50 * time.Millisecond)
	if got := calls.Load(); got != countAtCancel {
		t.Fatalf("calls kept increasing after cancellation: %d -> %d", countAtCancel, got)
	}
}

func TestRunOnSchedule_AsyncModeDoesNotBlockTheCaller(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	returned := make(chan struct{})
	go func() {
		// A long interval: if Async blocked like Sync does, this
		// goroutine would not reach the close(returned) below until the
		// first tick, which never happens within the test's timeout.
		runOnSchedule(ctx, time.Hour, ExecutionAsync, func() {})
		close(returned)
	}()

	select {
	case <-returned:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("runOnSchedule with ExecutionAsync blocked the calling goroutine")
	}
}

func TestRunOnSchedule_SyncModeBlocksUntilCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	returned := make(chan struct{})
	go func() {
		runOnSchedule(ctx, 5*time.Millisecond, ExecutionSync, func() {})
		close(returned)
	}()

	select {
	case <-returned:
		t.Fatal("runOnSchedule with ExecutionSync returned before the context was cancelled")
	case <-time.After(50 * time.Millisecond):
	}

	cancel()
	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("runOnSchedule with ExecutionSync did not return after cancellation")
	}
}
