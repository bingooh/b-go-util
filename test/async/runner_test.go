package async

import (
	"b-go-util/async"
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

//自定义任务，实现Run()
type MyTask struct {
	result int64
}

func (t *MyTask) Run(ctx context.Context) (<-chan struct{}, error) {
	//Run()仅做任务初始化，真正的业务处理在async.RunCancelableInterval()执行
	return async.RunCancelableInterval(ctx, 1*time.Second, func(c async.Context) {
		if c.Done() {
			fmt.Println("my task done: ", c.Count())
			return
		}

		t.result = c.Count() //设置任务结果
		fmt.Println("my task do: ", c.Count())
	}), nil
}

func TestRunnerBgTask(t *testing.T) {
	//runner负责执行传入的BgTask任务，同时提供Start()/Stop()/Wait()等帮助方法
	//runner适合与async.RunXX()配合使用，以下测试用例请谨慎考虑用于实际开发

	//BgTask任务不应阻塞调用的协程，否则runner.Start()将一直阻塞
	runner := async.NewRunner(async.ToBgTaskFn(func(ctx context.Context) (<-chan struct{}, error) {
		done := make(chan struct{})

		//阻塞调用协程，将导致runner.Start()阻塞
		fmt.Println("task1 enter:", time.Now().Second())
		time.Sleep(2 * time.Second)
		fmt.Println("task1 start", time.Now().Second())

		//启用1个后台协程执行任务
		go func() {
			//任务执行完成后关闭done，以通知runner
			defer close(done)

			fmt.Println("task1 do:", time.Now().Second())
			time.Sleep(2 * time.Second)
			fmt.Println("task1 done:", time.Now().Second())
		}()

		return done, nil
	}))

	start := time.Now()
	runner.MustStart() //这里将被阻塞，任务启动耗时2秒
	require.WithinDuration(t, time.Now(), start.Add(2*time.Second), 1*time.Second)

	start = time.Now()
	runner.Wait() //应在runner启动后调用，任务耗时3秒。任务自行关闭，不需调用runner.Stop()
	require.WithinDuration(t, time.Now(), start.Add(2*time.Second), 1*time.Second)

	//调用runner.Stop()将导致取消传给BgTask的ctx，BgTask应在监听到ctx取消后，关闭返回的管道
	//如果不关闭，则调用runner.Stop()将被一直阻塞直到BgTask完成
	runner = async.NewRunner(async.ToBgTaskFn(func(ctx context.Context) (<-chan struct{}, error) {
		done := make(chan struct{})

		go func() {
			<-ctx.Done() //应在ctx.Done()后关闭返回的done管道，以让runner正常退出
			fmt.Println("task2 ctx done:", time.Now().Second())

			//等待2秒后关闭done，将导致runner.Stop()阻塞2秒
			time.Sleep(2 * time.Second)
			close(done)

			fmt.Println("task2 done:", time.Now().Second())
		}()

		return done, nil
	}))
	runner.MustStart()

	start = time.Now()
	runner.Stop() //这里将被阻塞，任务关闭耗时2秒
	require.WithinDuration(t, time.Now(), start.Add(2*time.Second), 1*time.Second)
}

func TestRunner(t *testing.T) {
	//任务执行3秒，因外部ctx超时取消导致任务退出
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	runner := async.NewRunner(async.ToBgTaskFn(func(ctx context.Context) (<-chan struct{}, error) {
		return async.RunCancelableInterval(ctx, 1*time.Second, func(ctx async.Context) {
			if ctx.Done() {
				fmt.Println("task1 done: ", ctx.Count())
				return
			}

			fmt.Println("task1 do: ", ctx.Count())
		}), nil
	}))
	runner.WithContext(ctx).MustStart() //传入外部ctx

	start := time.Now()
	runner.Wait()
	require.WithinDuration(t, time.Now(), start.Add(3*time.Second), 1*time.Second)

	//自定义任务MyTask，实现BgTask接口，保存任务执行结果
	task := new(MyTask)
	runner = async.NewRunner(task)
	runner.WithAfterStop(func(ctx context.Context) {
		fmt.Println("my task after stop result: ", task.result)
	})

	runner.MustStart()
	time.Sleep(2 * time.Second)
	runner.Stop()
	require.True(t, task.result > 0)
}
