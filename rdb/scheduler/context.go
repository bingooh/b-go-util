package scheduler

import (
	"context"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"time"
)

type Context interface {
	Context() context.Context
	Logger() *zap.Logger
	Option() *TaskOption
	GetTaskNextInvokeTime() (time.Time, error)
	SetTaskNextInvokeTime(tm time.Time) error
}

type BaseContext struct {
	ctx    context.Context
	client *redis.Client
	logger *zap.Logger
	option *TaskOption
}

var _ Context = (*BaseContext)(nil)

func (c *BaseContext) Context() context.Context {
	return c.ctx
}

func (c *BaseContext) Logger() *zap.Logger {
	return c.logger
}

func (c *BaseContext) Option() *TaskOption {
	return c.option
}

func (c *BaseContext) GetTaskNextInvokeTime() (time.Time, error) {
	v, err := c.client.Get(c.ctx, c.option.TaskInvokeTimeKey()).Int64()
	if err != nil {
		if err == redis.Nil {
			err = nil
		}

		return time.Time{}, err
	}

	return time.Unix(v, 0), nil
}

func (c *BaseContext) SetTaskNextInvokeTime(tm time.Time) error {
	if tm.IsZero() {
		return c.client.Del(c.ctx, c.option.TaskInvokeTimeKey()).Err()
	}

	return c.client.Set(c.ctx, c.option.TaskInvokeTimeKey(), tm.Unix(), 0).Err()
}
