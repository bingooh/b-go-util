package async

import (
	"context"
	"github.com/bingooh/b-go-util/util"
	"time"
)

// 定时任务执行帮助类。支持设置首次执行延时，最大重试次数等
type RunIntervalHelper struct {
	interval      time.Duration //定时多久执行1次任务
	timeout       time.Duration //等待多久超时后退出执行任务
	initRunDelay  time.Duration //首次延迟多长时间执行任务，默认与interval相同
	maxRetryCount int64         //最大重试次数，默认为-1，表示不限制

	externalCtx context.Context //外部传入的ctx
}

// 创建定时任务执行帮助类，传入定时间隔
func NewRunIntervalHelper(interval time.Duration) *RunIntervalHelper {
	util.AssertOk(interval > 0, "interval <= 0")

	return &RunIntervalHelper{
		interval:      interval,
		initRunDelay:  interval,
		maxRetryCount: -1,
	}
}

// 设置任务执行上下文
func (r *RunIntervalHelper) WithContext(ctx context.Context) *RunIntervalHelper {
	r.externalCtx = ctx
	return r
}

// 设置任务超时时间
func (r *RunIntervalHelper) WithTimeout(timeout time.Duration) *RunIntervalHelper {
	util.AssertOk(timeout >= 0, "timeout < 0")
	r.timeout = timeout
	return r
}

// 设置任务首次执行延时时长
func (r *RunIntervalHelper) WithInitRunDelay(delay time.Duration) *RunIntervalHelper {
	util.AssertOk(delay >= 0, "delay < 0")
	r.initRunDelay = delay
	return r
}

// 设置任务最大重试次数。任务最多执行n+1次
func (r *RunIntervalHelper) WithMaxRetryCount(n int64) *RunIntervalHelper {
	util.AssertOk(n >= 0, "retry < 0")
	r.maxRetryCount = n
	return r
}

// 执行任务
func (r *RunIntervalHelper) Run(task func(ctx Context)) <-chan struct{} {
	ctx := r.externalCtx
	if ctx == nil {
		ctx = context.Background()
	}

	cancel := func() {}
	if r.timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, r.timeout)
	}

	retry := r.maxRetryCount

	return runCancelableInterval(ctx, r.interval, r.initRunDelay, func(ctx Context) {
		task(ctx)

		if !ctx.Done() && retry > 0 {
			retry--
			return
		}

		if !ctx.Done() && retry == 0 {
			ctx.Abort()
		}

		if ctx.Done() {
			cancel() //释放资源
		}
	})
}
