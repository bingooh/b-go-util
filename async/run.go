package async

import (
	"context"
	"sync"
	"time"
)

func handle(pool Pool, job func()) {
	if pool == nil {
		go job()
	} else {
		pool.Submit(job)
	}
}

func run(pool Pool, task func()) <-chan struct{} {
	done := make(chan struct{})

	handle(pool, func() {
		defer close(done)
		task()
	})

	return done
}

func runCancelable(ctx context.Context, pool Pool, task func()) <-chan struct{} {
	done := make(chan struct{})

	handle(pool, func() {
		defer close(done)

		select {
		case <-ctx.Done():
		case <-run(nil, task):
		}
	})

	return done
}

func runTask(pool Pool, task Task) <-chan Result {
	done := make(chan Result, 1)

	handle(pool, func() {
		defer close(done)

		done <- task.Run()
	})

	return done
}

func runCancelableTask(pool Pool, ctx context.Context, task Task) <-chan Result {
	done := make(chan Result, 1)

	handle(pool, func() {
		defer close(done)

		select {
		case <-ctx.Done():
			done <- NewResultWithContext(ctx)
		case r := <-runTask(nil, task):
			done <- r
		}
	})

	return done
}

func runCancelableInterval(pool Pool, ctx context.Context, interval, initRunDelay time.Duration, task func(ctx Context)) <-chan struct{} {
	return run(pool, func() {
		if initRunDelay <= 0 {
			initRunDelay = 1 * time.Nanosecond //避免创建ticker出错
		}

		//如果两者相同，则不需要再次创建ticker
		hasInitRun := initRunDelay == interval

		c := NewCtx(ctx)
		ticker := time.NewTicker(initRunDelay)
		for {
			c.(*BaseCtx).incrCount()

			select {
			case <-ctx.Done():
				ticker.Stop()
				task(c)
				return
			case <-ticker.C:
				//如果ticker触发很快，而task执行很慢，会不会有内存泄露？考虑使用timer?

				//执行task()时，ctx可能取消，导致c.Done()为true
				//最终可能导致task()的if c.Done(){}执行2次
				//以下替换为FreezeCtx
				task(newFreezeCtx(c, false))

				if c.Aborted() {
					ticker.Stop()
					return
				}

				if !hasInitRun {
					hasInitRun = true
					ticker.Stop()
					ticker = time.NewTicker(interval)
				}
			}
		}
	})
}

//执行任务
func Run(task func()) <-chan struct{} {
	return run(nil, task)
}

//执行可取消任务
func RunCancelable(ctx context.Context, task func()) <-chan struct{} {
	return runCancelable(ctx, nil, task)
}

//执行可超时任务
func RunTimeLimit(timeout time.Duration, task func()) <-chan struct{} {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	return runCancelable(ctx, nil, func() {
		defer cancel()
		task()
	})
}

//执行任务
func RunTask(task Task) <-chan Result {
	return runTask(nil, task)
}

//执行可取消任务
func RunCancelableTask(ctx context.Context, task Task) <-chan Result {
	return runCancelableTask(nil, ctx, task)
}

//执行可超时任务
func RunTimeLimitTask(timeout time.Duration, task Task) <-chan Result {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	return runCancelableTask(nil, ctx, TaskFn(func() Result {
		defer time.AfterFunc(1*time.Nanosecond, cancel) //避免过快取消导致执行<-ctx.Done()
		return task.Run()
	}))
}

//确保传入的任务已运行
func EnsureRun(tasks ...func()) {
	wg := &sync.WaitGroup{}
	wg.Add(len(tasks))

	for _, task := range tasks {
		go func(task func()) {
			wg.Done()
			task()
		}(task)
	}

	wg.Wait()
}

//确保传入的任务已全部执行完成
func EnsureDone(tasks ...func()) {
	wg := &sync.WaitGroup{}
	wg.Add(len(tasks))

	for _, task := range tasks {
		go func(task func()) {
			defer wg.Done()
			task()
		}(task)
	}

	wg.Wait()
}

// 循环执行任务直到取消
// 在多协程互相协作情况下，如果需要监听到ctx.Done()时立刻退出，
// 建议使用for-select，否则task可能多执行1次。这是因为前者阻塞在task，后者阻塞在select
func RunUtilCancel(ctx context.Context, task func(c Context)) <-chan struct{} {
	return Run(func() {
		DoUntilCancel(ctx, task)
	})
}

//循环执行任务直到超时
func RunUtilTimeout(timeout time.Duration, task func(c Context)) <-chan struct{} {
	return Run(func() {
		DoUntilTimeout(timeout, task)
	})
}

//执行可取消定时任务
func RunCancelableInterval(ctx context.Context, interval time.Duration, task func(ctx Context)) <-chan struct{} {
	return runCancelableInterval(nil, ctx, interval, interval, task)
}

//执行可超时定时任务
func RunTimeLimitInterval(interval, timeout time.Duration, task func(ctx Context)) <-chan struct{} {
	c, cancel := context.WithTimeout(context.Background(), timeout)
	return RunCancelableInterval(c, interval, func(ctx Context) {
		task(ctx)

		if ctx.Done() {
			cancel()
		}
	})
}
