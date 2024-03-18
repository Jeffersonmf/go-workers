package workermanager

import (
	"context"
	"time"

	"go-workers/internal/util"

	"go.uber.org/zap"
)

type Worker struct {
	SourceContext       context.Context
	Instrumentation     Instrumentation
	Callback            func(ctx context.Context)
	ExecsPerTick        []int
	TickDuration        time.Duration
	CronSchedulerConfig CronSchedulerConfig
}

var workerError WorkerError

func (wr Worker) Run() {
	if wr.CronSchedulerConfig.IntervalInSeconds > 0 {
		AddNewCronExecution(wr)
	} else {
		wr.blockToRun()
	}
}

func (wr Worker) blockToRun() {
	ctx := wr.SourceContext

	if wr.TickDuration <= 0 || wr.ExecsPerTick == nil {
		start := time.Now()
		wr.doWorkContextWithoutTickDuration()
		elapsed := time.Since(start)
		util.Sugar.Infof("Worker", wr.Instrumentation.FuncName, zap.String("elapsed time", elapsed.String()))
	} else {

		execs := wr.ExecsPerTick
		execsLen := len(execs)

		for i := 0; i < execsLen; i++ {
			// Do work
			start := time.Now()
			wr.Instrumentation.TaskArguments.SetIntParam("currentTick", i)
			wr.Instrumentation.TaskArguments.SetIntParam("ticksLeft", execsLen-i)
			wr.doWorkContext(
				execs[i],
			)
			elapsed := time.Since(start)
			if elapsed > wr.TickDuration {
				util.Sugar.Warnf(
					"Worker %s: on tick %d took %s to execute, which is more than the tick duration of %s",
					wr.Instrumentation.FuncName,
					i,
					elapsed,
					wr.TickDuration,
				)
			} else {
				time.Sleep(wr.TickDuration - elapsed)
			}

		}
	}

	if wr.Callback != nil {
		wr.Callback(ctx)
	}
}

func (wr *Worker) doWorkContext(execsCount int) {
	executionTries := 1
	chPostExecutionFail := make(chan bool, 1)

	executor := func(ctx context.Context, taskArg TaskParams) {
		//start := time.Now()
		callbackResult, err := wr.Instrumentation.FuncDispatcher(
			ctx,
			taskArg)

		//end := time.Now()
		//executionTime := end.Sub(start)
		currentTick, _ := taskArg.GetIntParam("currentTick")

		if err != nil {

			// Registering in the Datadog the error happened...
			workerError.RegisterMetricsCount(wr.Instrumentation.FuncName, 1, nil)

			util.Sugar.Errorf(
				"Failed to execute function %s on tick %d: %v",
				wr.Instrumentation.FuncName,
				currentTick,
				err,
			)
			chPostExecutionFail <- true
		} else {

			if wr.Instrumentation.NestedCallback != nil {
				err := wr.Instrumentation.NestedCallback(
					ctx,
					callbackResult)
				if err != nil {
					util.Sugar.Errorf("Failed to call nested callback")
				}
			}
		}
	}

	for i := 0; i < execsCount; i++ {
		taskArg := wr.Instrumentation.TaskArguments.Clone()
		taskArg.SetIntParam("currentExec", i)
		go executor(wr.SourceContext, taskArg)

		select {
		case <-chPostExecutionFail:
			if executionTries <= 3 {
				go executor(wr.SourceContext, taskArg)
				executionTries++
			}
		default:
			continue
		}
	}
}

func (wr *Worker) doWorkContextWithoutTickDuration() {
	// executionTries := 1
	chPostExecutionFail := make(chan bool, 1)

	executor := func(ctx context.Context, taskArg TaskParams) {
		//start := time.Now()
		callbackResult, err := wr.Instrumentation.FuncDispatcher(
			ctx,
			taskArg)

		//end := time.Now()
		//executionTime := end.Sub(start)

		if err != nil {

			// Registering in the Datadog the error happened...
			workerError.RegisterMetricsCount(wr.Instrumentation.FuncName, 1, nil)

			util.Sugar.Errorf(
				"Failed to execute function %s with error: %v",
				wr.Instrumentation.FuncName,
				err,
			)
			chPostExecutionFail <- true
		} else {

			if wr.Instrumentation.NestedCallback != nil {
				err := wr.Instrumentation.NestedCallback(
					ctx,
					callbackResult)
				if err != nil {
					util.Sugar.Errorf("Failed to call nested callback")
				}
			}
		}
	}

	taskArg := wr.Instrumentation.TaskArguments.Clone()
	go executor(wr.SourceContext, taskArg)

	select {
	case <-chPostExecutionFail:
		//TODO: Test later this part og the retries logic.
		// if executionTries <= 3 {
		util.Sugar.Errorf("****** A Critical Error is happened, in the Retries Zone!!!!! *******")
		go executor(wr.SourceContext, taskArg)
		// executionTries++
		// }
	default:
	}
}
