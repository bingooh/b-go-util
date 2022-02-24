package async

import (
	"context"
	"github.com/bingooh/b-go-util/util"
	"sync"
	"time"
)

//任务组，执行一组任务并保存其执行结果
// 注意,目前实现不支持:
// - 协程1添加任务，协程2等待任务执行完成
// - 当前任务执行时，向同1个组里添加新的任务
type Group struct {
	lock sync.Mutex
	wg   sync.WaitGroup

	existTaskCount *util.AtomicInt64 //已添加的任务数
	doneTaskCount  *util.AtomicInt64 //已完成的任务数

	pool   Pool
	result *GroupResult
}

func NewGroup() *Group {
	return &Group{
		existTaskCount: util.NewAtomicInt64(0),
		doneTaskCount:  util.NewAtomicInt64(0),
		result:         newGroupResult(),
	}
}

//设置Group执行任务使用的协程池
func (g *Group) WithPool(pool Pool) *Group {
	util.AssertOk(g.existTaskCount.Value() == 0, "group is running")
	g.pool = pool
	return g
}

//已添加的任务数
func (g *Group) ExistTaskCount() int64 {
	return g.existTaskCount.Value()
}

//已完成的任务数
func (g *Group) DoneTaskCount() int64 {
	return g.doneTaskCount.Value()
}

//待完成的任务数
func (g *Group) PendingTaskCount() int64 {
	return g.existTaskCount.Value() - g.doneTaskCount.Value()
}

//执行任务
func (g *Group) Run(task func()) {
	g.RunTask(ToVoidTask(task))
}

//执行任务
func (g *Group) RunTask(task Task) {
	g.wg.Add(1)
	i := g.existTaskCount.Incr(1) - 1

	handle(g.pool, func() {
		defer g.wg.Done()
		g.collect(i, task.Run())
	})
}

//执行可取消任务
func (g *Group) RunCancelableTask(ctx context.Context, task Task) {
	g.wg.Add(1)
	i := g.existTaskCount.Incr(1) - 1

	handle(g.pool, func() {
		defer g.wg.Done()

		select {
		case <-ctx.Done():
			g.collect(i, NewResultWithContext(ctx))
		case r := <-runTask(g.pool, task):
			g.collect(i, r)
		}
	})

}

//执行可超时任务
func (g *Group) RunTimeLimitTask(timeout time.Duration, task Task) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	g.RunCancelableTask(ctx, TaskFn(func() Result {
		defer time.AfterFunc(1*time.Nanosecond, cancel)
		return task.Run()
	}))
}

//等待任务执行完成并获取执行结果，多次调用将获取相同的执行结果
func (g *Group) Wait() *GroupResult {
	g.wg.Wait()
	return g.result
}

//等待任务执行完成或ctx.Done()取消等待
//注意：如果因ctx.Done()导致结束等待，后续完成的任务执行结果将不会添加到Group里
func (g *Group) WaitOrCancel(ctx context.Context) *GroupResult {
	c := DoCancelableTask(ctx, g.wg.Wait)

	g.result.cancel(c)
	return g.result
}

//等待任务执行完成或超时取消等待(ctx.Done())
//注意：如果因超时导致结束等待，后续完成的任务执行结果将不会添加到Group里
func (g *Group) WaitOrTimeout(timeout time.Duration) *GroupResult {
	c := DoTimeLimitTask(timeout, g.wg.Wait)

	g.result.cancel(c)
	return g.result
}

//收集任务执行结果
func (g *Group) collect(i int64, result Result) {
	g.doneTaskCount.Incr(1)
	g.result.put(int(i), result)
}
