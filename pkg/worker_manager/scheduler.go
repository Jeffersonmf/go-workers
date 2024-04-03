package workermanager

import (
	"fmt"
	"time"

	"github.com/Jeffersonmf/go-workers/v3/pkg/util"
	"github.com/go-co-op/gocron"
)

type TypeOfExecution int64

var TypeOfExecutionEnum = struct {
	Async TypeOfExecution
	Sync  TypeOfExecution
}{Async: 0,
	Sync: 1}

type CronSchedulerConfig struct {
	IntervalInSeconds int
	TypeOfExecution   TypeOfExecution
}

func init() {
}

func AddNewCronExecution(wr Worker) {

	var scheduler = gocron.NewScheduler(time.UTC)

	// Every starts the job immediately and then runs at the
	// specified interval
	job, err := scheduler.Every(wr.CronSchedulerConfig.IntervalInSeconds).Seconds().Do(func() {
		wr.blockToRun()
	})
	if err != nil {
		util.Sugar.Infof(err.Error())
	}

	job.Name(fmt.Sprintf("Job Scheduled at %s", time.Now().String()))

	switch wr.CronSchedulerConfig.TypeOfExecution {
	case TypeOfExecutionEnum.Async:
		scheduler.StartAsync()
	case TypeOfExecutionEnum.Sync:
		scheduler.StartBlocking()
	default:
		scheduler.StartAsync()
	}
}

func StopCronExecution(wr *Worker, scheduler *gocron.Scheduler) {
	wr.CronSchedulerConfig.IntervalInSeconds = -1
	scheduler.StopBlockingChan()
}

func MakeSchedulerAnchorPoint() {
	var schedulerInternal = gocron.NewScheduler(time.UTC)
	job, err := schedulerInternal.Every(10).Milliseconds().Do(func() {
	})
	if err != nil {
		util.Sugar.Infof(err.Error())
	}
	job.Name(fmt.Sprintf("Internal Block Job had been created at %s", time.Now().String()))
	util.Sugar.Infof(fmt.Sprintf("Internal Block Job had been created at %s", time.Now().String()))
	schedulerInternal.StartBlocking()
}
