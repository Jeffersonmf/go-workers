package workermanager

import (
	"errors"

	"go-workers/internal/util"
)

type WorkerError struct {
	msg string
}

type ExecutionException interface {
	RegisterMetricsCount(errorName string, count int64) error
}

func (e *WorkerError) RegisterMetricsCount(
	errorName string,
	count int64,
	errorTags []string,
) {
	e.msg = errorTags[0]
}

func (e *WorkerError) Error() string {
	return e.msg
}

func (e WorkerError) ExecutionError(args []string) *error {
	return new(error)
}

func WorkerErrorInstance(error string) *WorkerError {
	return &WorkerError{msg: error}
}

func CustomErrorInstance() error {
	return errors.New("not implemented")
}

func (w WorkerError) ListenErrosHappned() {
	// defer makes the function run at the end
	defer func() { // recovers panic
		if e := recover(); e != nil {
			util.Sugar.Infof(w.msg)
		}
	}()
}
