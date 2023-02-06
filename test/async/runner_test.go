package async

import (
	"context"
	"fmt"
	"github.com/bingooh/b-go-util/async"
	"github.com/bingooh/b-go-util/util"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// 自定义任务，实现Run()
type MyTask struct {
	result int64
}

func (t *MyTask) Run(ctx context.Context) <-chan struct{} {
	//Run()仅做任务初始化，真正的业务处理在async.RunCancelableInterval()执行
	return async.RunCancelableInterval(ctx, 1*time.Second, func(c async.Context) {
		if c.Done() {
			fmt.Println("my task done: ", c.Count())
			return
		}

		t.result = c.Count() //设置任务结果
		fmt.Println("my task do: ", c.Count())
	})
}

func TestRunnerBgTask(t *testing.T) {
	r := require.New(t)
	//runner负责执行传入的BgTask任务，同时提供Start()/Stop()/Wait()等帮助方法
	//runner适合与async.RunXX()配合使用，以下测试用例请谨慎考虑用于实际开发

	runner := async.NewTaskRunner(async.BgTaskFn(func(ctx context.Context) <-chan struct{} {
		done := make(chan struct{})

		//阻塞调用协程，不会导致runner.Start()阻塞
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

		return done
	}))

	start := time.Now()
	runner.Start() //不会阻塞，任务在后台执行
	r.True(runner.IsRunning())
	r.WithinDuration(time.Now(), start, 100*time.Millisecond)

	//任务耗时4秒后执行结束自动关闭，不需调用runner.Stop()
	time.Sleep(4100 * time.Millisecond)
	r.False(runner.IsRunning())

	start = time.Now()
	runner.Stop() //不会阻塞，任务已经结束
	r.WithinDuration(time.Now(), start, 100*time.Millisecond)

	//调用runner.Stop()将取消传给BgTask的ctx，BgTask监听到ctx取消后，
	//应关闭返回的管道，如果不关闭，则调用runner.Stop()将被一直阻塞直到BgTask完成
	runner = async.NewTaskRunner(async.BgTaskFn(func(ctx context.Context) <-chan struct{} {
		done := make(chan struct{})

		go func() {
			<-ctx.Done() //应在ctx.Done()后关闭返回的done管道，以让runner正常退出
			fmt.Println("task2 ctx done:", time.Now().Second())

			//等待2秒后关闭done，将导致runner.Stop()阻塞2秒
			time.Sleep(2 * time.Second)
			close(done)

			fmt.Println("task2 done:", time.Now().Second())
		}()

		return done
	}))
	runner.Start()

	start = time.Now()
	runner.Stop() //阻塞直到任务关闭返回的管道，耗时2秒
	r.WithinDuration(time.Now(), start.Add(2*time.Second), 1*time.Second)

	runner = async.NewTaskRunner(async.BgTaskFn(func(ctx context.Context) <-chan struct{} {
		done := make(chan struct{})

		go func() {
			timer := time.NewTimer(2 * time.Second)
			defer func() {
				timer.Stop()
				close(done)
			}()

			select {
			case <-ctx.Done():
				fmt.Println(`task3 ctx done`)
			case <-timer.C:
				fmt.Println(`task3 done`)
			}
		}()

		return done
	}))
	runner.Start()

	start = time.Now()
	runner.Wait() //等等任务执行完成，耗时2秒，不会取消ctx导致任务退出
	r.False(runner.IsRunning())
	r.WithinDuration(time.Now(), start.Add(2*time.Second), 100*time.Millisecond)
}

func TestRunner(t *testing.T) {
	r := require.New(t)

	//任务执行3秒，因外部ctx超时取消导致任务退出
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	runner := async.NewTaskRunner(async.BgTaskFn(func(ctx context.Context) <-chan struct{} {
		return async.RunCancelableInterval(ctx, 1*time.Second, func(ctx async.Context) {
			if ctx.Done() {
				fmt.Println("task1 done: ", ctx.Count())
				return
			}

			fmt.Println("task1 do: ", ctx.Count())
		})
	}))

	runner.WithContext(ctx).Start() //传入外部ctx
	time.Sleep(3100 * time.Millisecond)
	r.False(runner.IsRunning())

	//自定义任务MyTask，实现BgTask接口，保存任务执行结果
	task := new(MyTask)
	runner = async.NewTaskRunner(task)

	runner.Start()
	time.Sleep(2 * time.Second)
	runner.Stop()
	r.False(runner.IsRunning())
	r.True(task.result > 0)

	//搭配RunXX()使用，任务执行3秒后主动关闭runner
	runner = async.NewTaskRunner(async.BgTaskFn(func(ctx context.Context) <-chan struct{} {
		return async.RunCancelableInterval(ctx, 1*time.Second, func(c async.Context) {
			if c.Done() {
				fmt.Println(`task3 done`)
				return
			}

			fmt.Println(`task3 do:`, time.Now().Format(`05`))
		})
	}))
	runner.Start()
	time.Sleep(3 * time.Second)
	runner.Stop()

	//任务执行3秒后结束，runner自动关闭
	runner = async.NewTaskRunner(async.BgTaskFn(func(ctx context.Context) <-chan struct{} {
		return async.RunCancelable(ctx, func() {
			time.Sleep(3 * time.Second)
		})
	}))
	runner.Start()
	r.True(runner.IsRunning())
	start := time.Now()
	runner.Wait()
	r.False(runner.IsRunning())
	r.WithinDuration(time.Now(), start.Add(3*time.Second), 100*time.Millisecond)
}

type MyRunner struct {
	i         int
	isRunning bool
	ctx       *util.CancelableContext
}

func NewMyRunner(i int) async.Runner {
	return &MyRunner{
		i:   i,
		ctx: util.NewCancelableContext(),
	}
}

func (r *MyRunner) Start() {
	if r.isRunning {
		return
	}
	r.isRunning = true

	tag := fmt.Sprintf(`task%v:`, r.i)
	async.RunCancelableInterval(r.ctx.Context(), time.Duration(r.i)*time.Second, func(c async.Context) {
		if c.Done() {
			r.isRunning = false
			fmt.Println(tag, `done`)
			return
		}

		fmt.Println(tag, time.Now().Format(`0405`))
	})
}

func (r *MyRunner) Stop() {
	r.ctx.Cancel()
}

func (r *MyRunner) IsRunning() bool {
	return r.isRunning
}

func TestRunnerGroup(t *testing.T) {
	g := async.NewRunnerGroup()
	g.Add(NewMyRunner(2))
	g.Start()

	r3 := NewMyRunner(3)
	r3.Start() //需要自行启动
	g.Add(r3)

	time.Sleep(3 * time.Second)
	r3.Stop() //需要自行关闭
	g.Del(r3)

	time.Sleep(3 * time.Second)
	g.Stop()
}
