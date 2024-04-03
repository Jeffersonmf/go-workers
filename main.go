package main

import (
	"context"

	"github.com/Jeffersonmf/go-workers/v3/pkg/util"
	workermanager "github.com/Jeffersonmf/go-workers/v3/pkg/worker_manager"
)

func init() {
}

func main() {
	util.Sugar.Infof("The Go-Workers module has been started.")

	ctx := context.Background()

	loadDataStep := func(ctx context.Context, taskArg workermanager.TaskParams) (workermanager.TaskParams, error) {
		println("Load Data from ReadShift Brand DNB table")
		return taskArg, nil
	}

	writeDataStep := func(ctx context.Context, taskArg workermanager.TaskParams) (workermanager.TaskParams, error) {
		return taskArg, nil
	}

	listenCallbackStep := func(ctx context.Context, taskArg workermanager.TaskParams) error {
		workermanager.Worker{

			SourceContext: ctx,
			Instrumentation: workermanager.Instrumentation{
				TaskArguments:  taskArg,
				FuncDispatcher: writeDataStep,
				FuncName:       "Step to Write data into Hotdata Database",
			},
		}.Run()

		return nil
	}

	workermanager.Worker{
		SourceContext: ctx,
		CronSchedulerConfig: workermanager.CronSchedulerConfig{
			IntervalInSeconds: 1,
			TypeOfExecution:   workermanager.TypeOfExecutionEnum.Sync,
		},
		Instrumentation: workermanager.Instrumentation{
			TaskArguments:  workermanager.NewTaskParams(),
			FuncDispatcher: loadDataStep,
			NestedCallback: listenCallbackStep,
			FuncName:       "Step to Listen Data from Redshift",
		},
	}.Run()

}
