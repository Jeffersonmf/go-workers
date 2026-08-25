package workermanager

import "fmt"

// TaskError wraps a task execution failure with the task's name and
// which attempt it was, so an OnError handler or a log line can
// identify what failed without parsing a formatted message.
//
// This replaced the previous WorkerError/ExecutionException pair: that
// interface declared RegisterMetricsCount(string, int64) error, but
// the only implementation had a different signature (three parameters,
// no return value) and so never actually satisfied the interface it
// was declared against. Nothing in the codebase called it through the
// interface either; it was dead weight that looked load-bearing.
type TaskError struct {
	TaskName string
	Attempt  int
	Err      error
}

func (e *TaskError) Error() string {
	return fmt.Sprintf("task %q failed on attempt %d: %v", e.TaskName, e.Attempt, e.Err)
}

func (e *TaskError) Unwrap() error {
	return e.Err
}
