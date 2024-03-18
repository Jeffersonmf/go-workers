package workermanager

import (
	"context"
)

type TaskParams struct {
	Params map[string]interface{}
}

func NewTaskParams() TaskParams {
	return TaskParams{
		Params: make(map[string]interface{}),
	}
}

func (t *TaskParams) GetParams() map[string]interface{} {
	return t.Params
}

func (t *TaskParams) SetStringParam(key string, value string) {
	t.Params[key] = value
}

func (t *TaskParams) SetIntParam(key string, value int) {
	t.Params[key] = value
}

func (t *TaskParams) SetFloatParam(key string, value float64) {
	t.Params[key] = value
}

func (t *TaskParams) SetBytesParam(key string, value []byte) {
	t.Params[key] = value
}

func (t *TaskParams) SetComplexParam(key string, value *any) {
	t.Params[key] = value
}

func (t *TaskParams) GetComplexParam(key string) (*any, bool) {
	value, ok := t.Params[key].(*any)
	return value, ok
}

func (t *TaskParams) GetBytesParam(key string) ([]byte, bool) {
	value, ok := t.Params[key].([]byte)
	return value, ok
}

func (t *TaskParams) GetStringParam(key string) (string, bool) {
	value, ok := t.Params[key].(string)
	return value, ok
}

func (t *TaskParams) GetIntParam(key string) (int, bool) {
	value, ok := t.Params[key].(int)
	return value, ok
}

func (t *TaskParams) GetFloatParam(key string) (float64, bool) {
	value, ok := t.Params[key].(float64)
	return value, ok
}

func (t *TaskParams) Clone() TaskParams {
	newParams := make(map[string]interface{})
	for key, value := range t.Params {
		newParams[key] = value
	}
	return TaskParams{
		Params: newParams,
	}
}

type Instrumentation struct {
	FuncDispatcher func(ctx context.Context, data TaskParams) (TaskParams, error)
	NestedCallback func(ctx context.Context, data TaskParams) error
	TaskArguments  TaskParams
	FuncName       string
}
