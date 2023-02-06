package scheduler

import (
	"context"
	"fmt"
	"github.com/bingooh/b-go-util/_string"
	"github.com/bingooh/b-go-util/util"
	"github.com/go-redis/redis/v8"
	"time"
)

type TaskOption struct {
	name      string //任务名称
	keyPrefix string //任务redis key前缀

	Disabled        bool          //是否禁用
	DisableTaskLock bool          //是否禁用任务锁
	TaskLockTTL     time.Duration //任务锁TTL，默认10s
	InvokeInternal  time.Duration //任务执行间隔时长，默认1m
	InvokeTimeRange []int         //任务执行时间区间，格式HHMI，如：[1200,1400]
}

func (o *TaskOption) TaskLockKey() string {
	return fmt.Sprintf(`%v:lock:%v`, o.keyPrefix, o.name)
}

func (o *TaskOption) TaskInvokeTimeKey() string {
	return fmt.Sprintf(`%v:invoke:%v`, o.keyPrefix, o.name)
}

func (o *TaskOption) MustNormalize() *TaskOption {
	util.AssertOk(o != nil, `option为空`)
	util.AssertNotEmpty(o.name, `name为空`)
	util.AssertNotEmpty(o.keyPrefix, `keyPrefix为空`)

	n := len(o.InvokeTimeRange)
	util.AssertOk(n == 0 || n == 2 && o.InvokeTimeRange[0] <= o.InvokeTimeRange[1], `无效InvokeTimeRange[%v]`, o.InvokeTimeRange)

	if o.TaskLockTTL <= 0 {
		o.TaskLockTTL = 10 * time.Second
	}

	if o.InvokeInternal <= 0 {
		o.InvokeInternal = 1 * time.Minute
	}

	return o
}

type Option struct {
	rootCtx *util.CancelableContext //由scheduler取消
	Redis   *redis.Options

	Tasks         map[string]*TaskOption //key为任务名称
	TaskKeyPrefix string                 //任务redis key前缀，默认task
}

func (o *Option) MustNormalize() *Option {
	util.AssertOk(o != nil, `option为空`)
	util.AssertOk(o.Redis != nil, `Redis为空`)
	util.AssertOk(len(o.Tasks) > 0, `Tasks为空`)

	o.rootCtx = util.NewCancelableContext()
	if _string.Empty(o.TaskKeyPrefix) {
		o.TaskKeyPrefix = `task:`
	}

	for name, to := range o.Tasks {
		util.AssertOk(to != nil, `TaskOption为空`)
		to.name = name
		to.keyPrefix = o.TaskKeyPrefix
		to.MustNormalize()
	}

	return o
}

func (o *Option) RootContext() context.Context {
	return o.rootCtx.Context()
}

func (o *Option) MustGetTaskOption(name string) *TaskOption {
	to, ok := o.Tasks[name]
	util.AssertOk(ok, `TaskOption为空[name=%v]`, name)
	return to
}
