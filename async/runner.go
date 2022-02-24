package async

import (
	"context"
	"github.com/bingooh/b-go-util/util"
	"sync"
)

//后台任务，添加给Runner执行
//接口实现需要保证：
//  - 不要阻塞当前调用协程，在Run()里应仅做任务初始化，然后自行启用后台协程执行真正的业务处理
//  - 任务初始化出错应返回error
//	- 任务执行完成后应关闭返回的管道
//  - 任务初始化时应监听ctx.Done(),在ctx取消后关闭返回的管道
//
//建议配合使用async.RunXX(),可参考示例
type BgTask interface {
	Run(ctx context.Context) (<-chan struct{}, error)
}

type BgTaskFn func(ctx context.Context) (<-chan struct{}, error)

func (f BgTaskFn) Run(ctx context.Context) (<-chan struct{}, error) {
	return f(ctx)
}

func ToBgTaskFn(fn func(ctx context.Context) (<-chan struct{}, error)) BgTaskFn {
	return BgTaskFn(fn)
}

//后台任务执行者，提供Start()/Stop()等帮助方法用于启动/停止后台任务
type Runner struct {
	ctx       context.Context
	ctxCancel func()

	lock    sync.Mutex
	running bool

	stopSign  sync.WaitGroup
	afterStop func(ctx context.Context)

	task     BgTask
	taskDone <-chan struct{}
}

//创建后台任务执行者，并添加要执行的后台任务
func NewRunner(task BgTask) *Runner {
	util.AssertOk(task != nil, "task is nil")
	return &Runner{task: task}
}

//设置任务执行上下文，此对象将传给BgTask。默认runner内部创建1个ctx
func (r *Runner) WithContext(ctx context.Context) *Runner {
	util.AssertOk(!r.Running(), "disallow invoke WithContext() when runner is running")
	r.ctx = ctx
	return r
}

//设置任务结束回调函数
func (r *Runner) WithAfterStop(fn func(ctx context.Context)) *Runner {
	util.AssertOk(!r.Running(), "disallow invoke WithAfterStop() when runner is running")
	r.afterStop = fn
	return r
}

//是否正在运行中
func (r *Runner) Running() bool {
	r.lock.Lock()
	defer r.lock.Unlock()

	return r.running
}

//开始执行任务，返回启动错误
func (r *Runner) Start() error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.running {
		return nil
	}

	if r.ctx == nil {
		r.ctx = context.Background()
	}
	r.ctx, r.ctxCancel = context.WithCancel(r.ctx)

	//如果task()阻塞线程，后续包括调用Stop都无法执行
	//考虑使用新协程或time.AfterFunc()等检查是否阻塞当前线程
	var err error
	if r.taskDone, err = r.task.Run(r.ctx); err != nil {
		return err
	}

	r.stopSign.Add(1)
	r.running = true

	//等待任务执行完成
	go func() {
		<-r.taskDone
		r.Stop()
	}()

	return nil
}

func (r *Runner) MustStart() *Runner {
	util.AssertNilErr(r.Start())
	return r
}

//停止执行任务
func (r *Runner) Stop() {
	if r == nil {
		return
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	if !r.running {
		return
	}

	r.ctxCancel()
	<-r.taskDone

	r.stopSign.Done()
	r.running = false

	if r.afterStop != nil {
		r.afterStop(r.ctx)
	}
}

//等待任务执行完成。应在启动后调用此方法，否则可能报错
func (r *Runner) Wait() {
	//没有判断r.running，避免竞争锁
	r.stopSign.Wait()
}
