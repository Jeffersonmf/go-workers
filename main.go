// Command go-workers is a runnable example of the workermanager
// package: a scheduled worker that fetches a batch of records from a
// source, hands each one off to a second step, and shuts down
// cleanly on SIGINT/SIGTERM instead of being killed mid-batch.
package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/Jeffersonmf/go-workers/pkg/util"
	workermanager "github.com/Jeffersonmf/go-workers/pkg/worker_manager"
)

type record struct {
	ID   int
	Name string
}

func main() {
	util.Logger.Info("go-workers example starting")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	fetchBatch := func(_ context.Context, _ workermanager.TaskParams) (workermanager.TaskParams, error) {
		batch := []record{{ID: 1, Name: "first"}, {ID: 2, Name: "second"}}

		result := workermanager.NewTaskParams()
		workermanager.SetParam(result, "batch", batch)
		return result, nil
	}

	persistBatch := func(_ context.Context, args workermanager.TaskParams) error {
		batch, ok := workermanager.GetParam[[]record](args, "batch")
		if !ok {
			return fmt.Errorf("expected a []record under \"batch\"")
		}

		for _, r := range batch {
			util.Logger.Info("persisted record", "id", r.ID, "name", r.Name)
		}
		return nil
	}

	workermanager.Worker{
		SourceContext: ctx,
		CronSchedulerConfig: workermanager.CronSchedulerConfig{
			IntervalInSeconds: 5,
			TypeOfExecution:   workermanager.TypeOfExecutionEnum.Sync,
		},
		Instrumentation: workermanager.Instrumentation{
			TaskArguments:  workermanager.NewTaskParams(),
			FuncDispatcher: fetchBatch,
			NestedCallback: persistBatch,
			FuncName:       "fetch-and-persist-batch",
		},
		OnError: func(err *workermanager.TaskError) {
			util.Logger.Error("worker step failed", "task", err.TaskName, "attempt", err.Attempt, "error", err.Err)
		},
	}.Run()

	util.Logger.Info("go-workers example stopped")
}
