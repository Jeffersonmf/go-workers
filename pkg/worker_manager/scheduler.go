package workermanager

import (
	"context"
	"time"
)

// TypeOfExecution selects whether a scheduled Worker blocks the
// calling goroutine (Sync) or runs in the background (Async).
type TypeOfExecution int

const (
	// ExecutionAsync runs the schedule in a background goroutine; the
	// caller of Worker.Run gets control back immediately.
	ExecutionAsync TypeOfExecution = iota
	// ExecutionSync blocks the calling goroutine until the Worker's
	// context is cancelled.
	ExecutionSync
)

// TypeOfExecutionEnum mirrors the previous struct-literal enum
// (TypeOfExecutionEnum.Async / TypeOfExecutionEnum.Sync) so it reads
// the same at call sites as before, backed now by real typed
// constants instead of package-level struct fields.
var TypeOfExecutionEnum = struct {
	Async TypeOfExecution
	Sync  TypeOfExecution
}{Async: ExecutionAsync, Sync: ExecutionSync}

// CronSchedulerConfig makes a Worker run on a fixed interval instead of
// once. IntervalInSeconds must be greater than zero for scheduling to
// take effect; see Worker.Run.
type CronSchedulerConfig struct {
	IntervalInSeconds int
	TypeOfExecution   TypeOfExecution
}

// runOnSchedule runs fn immediately, then again every interval, until
// ctx is cancelled. There is no separate Stop function: ctx
// cancellation is already how the rest of this package propagates
// shutdown (a Worker's SourceContext flows into every task it runs),
// so the scheduler follows that same convention instead of
// introducing a second, unrelated way to stop things.
//
// This replaced go-co-op/gocron (and its transitive dependency on a
// second cron implementation, robfig/cron), which existed only to do
// "run a function every N seconds" — exactly what the standard
// library's time.Ticker already does, with the benefit of composing
// naturally with context cancellation instead of needing its own
// Stop mechanism.
func runOnSchedule(ctx context.Context, interval time.Duration, execType TypeOfExecution, fn func()) {
	run := func() {
		fn()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				fn()
			}
		}
	}

	if execType == ExecutionAsync {
		go run()
		return
	}
	run()
}
