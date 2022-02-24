package async

import (
	"context"
	"time"
)

//循环执行任务直到取消
func DoUntilCancel(ctx context.Context, task func(c Context)) {
	c := NewCtx(ctx)
	for {
		c.(*BaseCtx).incrCount()

		select {
		case <-ctx.Done():
			task(c)
			return
		default:
			task(newFreezeCtx(c, false))

			if c.Aborted() {
				return
			}
		}
	}
}

//循环执行任务直到超时
func DoUntilTimeout(timeout time.Duration, task func(c Context)) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	DoUntilCancel(ctx, task)
}

//执行可取消任务
func DoCancelableTask(ctx context.Context, task func()) Context {
	c := NewCtx(ctx)
	select {
	case <-ctx.Done():
		return c
	case <-run(nil, task):
		return newFreezeCtx(c, false)
	}
}

//执行可超时任务
func DoTimeLimitTask(timeout time.Duration, task func()) Context {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer time.AfterFunc(1*time.Nanosecond, cancel)
	return DoCancelableTask(ctx, task)
}
