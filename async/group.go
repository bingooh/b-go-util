package async

import (
	"github.com/bingooh/b-go-util/util"
	"sync"
)

type WaitGroup struct {
	wg sync.WaitGroup
}

func NewWaitGroup() *WaitGroup {
	return &WaitGroup{}
}

func (g *WaitGroup) Run(task func()) {
	g.wg.Add(1)

	go func() {
		defer g.wg.Done()
		task()
	}()
}

// Wait 等待任务执行完成。调用此方法后再调用g.Run()添加新任务可能触发崩溃错误
func (g *WaitGroup) Wait() {
	g.wg.Wait()
}

// WorkerGroup 多协程执行不同任务
type WorkerGroup struct {
	wg     sync.WaitGroup
	permit chan struct{} //许可证
}

func NewWorkerGroup(limit int) *WorkerGroup {
	util.AssertOk(limit > 0, `limit<=0`)

	g := &WorkerGroup{
		permit: make(chan struct{}, limit),
	}

	return g
}

func (g *WorkerGroup) Run(task func()) {
	g.wg.Add(1)
	g.permit <- struct{}{}

	go func() {
		defer func() {
			<-g.permit
			g.wg.Done()
		}()

		task()
	}()
}

// Wait 等待任务执行完成。调用此方法后再调用g.Run()添加新任务可能触发崩溃错误
func (g *WorkerGroup) Wait() {
	g.wg.Wait()
}

// RoutineGroup 多协程执行同1个任务
type RoutineGroup struct {
	wg    sync.WaitGroup
	task  func()
	limit int
}

func NewRoutineGroup(limit int, task func()) *RoutineGroup {
	util.AssertOk(limit > 0, `limit<=0`)
	util.AssertOk(task != nil, `task为空`)
	return &RoutineGroup{limit: limit, task: task}
}

func (g *RoutineGroup) Start() {
	g.wg.Add(g.limit)
	for i := 0; i < g.limit; i++ {
		go func() {
			defer g.wg.Done()
			g.task()
		}()
	}
}

func (g *RoutineGroup) Run() {
	g.Start()
	g.Wait()
}

// Wait 等待任务执行完成。调用此方法后再调用g.Start()可能触发崩溃错误
func (g *RoutineGroup) Wait() {
	g.wg.Wait()
}
