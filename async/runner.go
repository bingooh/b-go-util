package async

import (
	"context"
	"github.com/bingooh/b-go-util/util"
	"sync"
	"time"
)

type Runner interface {
	Start()
	Stop()
	IsRunning() bool
}

// BgTask 后台任务，执行完成应关闭返回的管道
// 接口实现必须满足以下条件：
//   - Run()不行长时间阻塞调用协程，建议此方法只做初始化，然后启动另1后台协程执行具体任务
//   - 任务初始化时应监听ctx.Done(),在ctx取消后关闭返回的管道
//   - 任务执行完成后应关闭返回的管道
//
// 建议配合使用async.RunXX(),可参考示例
type BgTask interface {
	Run(ctx context.Context) <-chan struct{}
}

type BgTaskFn func(ctx context.Context) <-chan struct{}

func (f BgTaskFn) Run(ctx context.Context) <-chan struct{} {
	return f(ctx)
}

// TaskRunner 任务执行者，用于执行BgTask
type TaskRunner struct {
	isRunning *util.AtomicBool
	taskCtx   *util.CancelableContext
	task      BgTask
	taskDone  sync.WaitGroup
}

func NewTaskRunner(task BgTask) *TaskRunner {
	util.AssertOk(task != nil, `task为空`)
	return &TaskRunner{
		task:      task,
		isRunning: util.NewAtomicBool(false),
	}
}

func (g *TaskRunner) WithContext(ctx context.Context) *TaskRunner {
	g.taskCtx.Cancel()
	g.taskCtx = util.NewCancelableContextWithParent(ctx)
	return g
}

func (g *TaskRunner) IsRunning() bool {
	return g.isRunning.Value()
}

func (g *TaskRunner) Start() {
	if !g.isRunning.CASwap(false) {
		return
	}

	g.taskDone.Add(1)

	if g.taskCtx == nil {
		g.taskCtx = util.NewCancelableContext()
	}

	go func() {
		<-g.task.Run(g.taskCtx.Context())
		g.taskDone.Done()
		g.Stop()
	}()
}

// Stop 取消任务ctx并等待任务结束
func (g *TaskRunner) Stop() {
	if !g.isRunning.CASwap(true) {
		return
	}

	g.taskCtx.Cancel()
	g.waitDone()
}

func (g *TaskRunner) waitDone() {
	time.Sleep(1 * time.Millisecond) //等等g.Start()执行完成g.taskDone.Add(1)
	g.taskDone.Wait()
}

// Wait 等待任务执行完成
func (g *TaskRunner) Wait() {
	if g.isRunning.True() {
		g.waitDone()
	}
}

type RunnerGroup struct {
	lock      sync.Mutex
	isRunning bool
	runners   map[Runner]uint8
}

func NewRunnerGroup() *RunnerGroup {
	return &RunnerGroup{
		runners: make(map[Runner]uint8),
	}
}

func (g *RunnerGroup) Start() {
	g.lock.Lock()
	defer g.lock.Unlock()

	for runner := range g.runners {
		runner.Start()
	}
}

func (g *RunnerGroup) Stop() {
	g.lock.Lock()
	defer g.lock.Unlock()

	for runner := range g.runners {
		runner.Stop()
	}
}

func (g *RunnerGroup) Add(runners ...Runner) *RunnerGroup {
	g.lock.Lock()
	defer g.lock.Unlock()

	for _, runner := range runners {
		g.runners[runner] = 1
	}

	return g
}

func (g *RunnerGroup) Del(runners ...Runner) *RunnerGroup {
	g.lock.Lock()
	defer g.lock.Unlock()

	for _, runner := range runners {
		if g.runners[runner] == 1 {
			delete(g.runners, runner)
		}
	}

	return g
}
