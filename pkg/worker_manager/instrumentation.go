// Package workermanager runs caller-defined task functions, optionally
// on a schedule and optionally as repeated ticks of concurrent
// executions, with retries and panic recovery around every execution.
package workermanager

import "context"

// TaskParams carries typed values between the steps of a task chain.
// Values are stored as `any` internally and accessed through the
// generic SetParam/GetParam functions, which replace what used to be
// fourteen near-identical Set<Type>Param/Get<Type>Param methods (one
// pair per supported type: string, int, float64, []byte, *sql.Rows,
// *any). Generics collapse that into two functions that work for any
// type, including ones the original code never anticipated.
type TaskParams struct {
	values map[string]any
}

// NewTaskParams returns an empty, ready-to-use TaskParams.
func NewTaskParams() TaskParams {
	return TaskParams{values: make(map[string]any)}
}

// SetParam stores value under key.
func SetParam[T any](t TaskParams, key string, value T) {
	t.values[key] = value
}

// GetParam returns the value stored under key, and whether it was
// present and had the requested type T. A key that exists but holds a
// different type reports ok=false, the same as a missing key: callers
// that only care about "do I have a usable value" do not need to
// distinguish the two cases.
func GetParam[T any](t TaskParams, key string) (T, bool) {
	raw, exists := t.values[key]
	if !exists {
		var zero T
		return zero, false
	}
	value, ok := raw.(T)
	return value, ok
}

// Params returns a copy of the underlying map, for callers that need
// to range over every stored value regardless of type.
func (t TaskParams) Params() map[string]any {
	out := make(map[string]any, len(t.values))
	for k, v := range t.values {
		out[k] = v
	}
	return out
}

// Clone returns a shallow copy of t: a new TaskParams with the same
// key/value pairs, safe to mutate without affecting the original. Each
// concurrent task execution clones the shared TaskArguments before
// running, so tasks never race on the same underlying map.
func (t TaskParams) Clone() TaskParams {
	clone := NewTaskParams()
	for k, v := range t.values {
		clone.values[k] = v
	}
	return clone
}

// Instrumentation describes one step of a task chain: the function to
// run (FuncDispatcher), an optional next step to run with its result
// (NestedCallback), the arguments to pass in, and a name used for logs
// and error reporting.
type Instrumentation struct {
	FuncDispatcher func(ctx context.Context, data TaskParams) (TaskParams, error)
	NestedCallback func(ctx context.Context, data TaskParams) error
	TaskArguments  TaskParams
	FuncName       string
}
