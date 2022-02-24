package async

import (
	"context"
	"github.com/bingooh/b-go-util/util"
	"sync"
	"time"
)

// 协程池
// 注意,目前实现不支持:
// - 协程1添加任务，协程2等待任务执行完成
// - 当前任务执行时，向同1个池里添加新的任务
type Pool interface {
	Wait()                                                                                                     //等待任务执行完成
	Close()                                                                                                    //等待任务执行完成后关闭
	CloseWithTimeout(t time.Duration)                                                                          //等待任务执行完成后关闭或超时后关闭
	PendingTaskSize() int64                                                                                    //待完成的的任务数
	ExistWorkerSize() int64                                                                                    //当前启用的协程数
	Submit(task func())                                                                                        //提交任务到协程池
	Run(task func()) <-chan struct{}                                                                           //执行任务
	RunCancelable(ctx context.Context, task func()) <-chan struct{}                                            //执行可取消任务
	RunTimeLimit(timeout time.Duration, task func()) <-chan struct{}                                           //执行可超时任务
	RunTask(task Task) <-chan Result                                                                           //执行任务
	RunCancelableTask(ctx context.Context, task Task) <-chan Result                                            //执行可取消任务
	RunTimeLimitTask(timeout time.Duration, task Task) <-chan Result                                           //执行可超时任务
	RunCancelableInterval(ctx context.Context, interval time.Duration, task func(ctx Context)) <-chan struct{} //执行可取消定时任务
	RunTimeLimitInterval(interval, timeout time.Duration, task func(ctx Context)) <-chan struct{}              //执行可超时定时任务
}

//协程池实现类
type BasePool struct {
	minWorkerSize    int64
	closed           *util.AtomicBool
	workerCount      *util.AtomicInt64 //已创建的协程数
	pendingTaskCount *util.AtomicInt64 //待完成的的任务数(正执行和待执行)
	quit             chan struct{}
	permit           chan struct{}
	taskQueue        chan func()

	allTaskDone       *util.AtomicBool
	allTaskDoneWg     *sync.WaitGroup
	allWorkerExitSign chan struct{}
}

//创建协程池
// minWorkerSize  最小协程数，协程池将创建此数量的协程，直到关闭时才销毁
// maxWorkerSize  最大协程数，协程池最多创建此数量的协程，其中(maxWorkerSize-minWorkerSize)的协程在空闲一段时间后销毁
// taskQueueSize  缓存任务的队列大小，添加任务时如果协程池的协程数已经达到maxWorkerSize，且没有空闲协程。则任务将添加到此队列里
func NewWorkerPool(minWorkerSize, maxWorkerSize, taskQueueSize int64) Pool {
	util.AssertOk(minWorkerSize > 0, "minWorkerSize <=0")
	util.AssertOk(minWorkerSize <= maxWorkerSize, "minWorkerSize > maxWorkerSize")
	//util.AssertOk(minWorkerSize <= taskQueueSize, "minWorkerSize > pendingTaskCount")

	b := &BasePool{
		closed:            util.NewAtomicBool(false),
		minWorkerSize:     minWorkerSize,
		workerCount:       util.NewAtomicInt64(0),
		pendingTaskCount:  util.NewAtomicInt64(0),
		quit:              make(chan struct{}),
		permit:            make(chan struct{}, maxWorkerSize),
		taskQueue:         make(chan func(), taskQueueSize),
		allTaskDoneWg:     &sync.WaitGroup{},
		allTaskDone:       util.NewAtomicBool(true),
		allWorkerExitSign: make(chan struct{}),
	}

	for i := int64(0); i < minWorkerSize; i++ {
		b.permit <- struct{}{} //获取许可
		b.startWorker()
	}

	return b
}

//待完成的的任务数
func (b *BasePool) PendingTaskSize() int64 {
	return b.pendingTaskCount.Value()
}

//当前启用的协程数
func (b *BasePool) ExistWorkerSize() int64 {
	return b.workerCount.Value()
}

//协程池是否已关闭
func (b *BasePool) Closed() bool {
	return b.closed.True()
}

//等待任务执行完成后关闭
//调用此方法后将丢弃任务队列待处理的任务，并且禁止提交新任务
func (b *BasePool) Close() {
	b.CloseWithTimeout(-1)
}

// 等待任务执行完成后关闭或超时后关闭
// 调用此方法后将丢弃任务队列待处理的任务，并且禁止提交新任务
// 如果参数t<0，则会一直等待直到所有任务执行完成，否则超时后关闭
// 注意：调用此方法不会停止正在执行任务的后台协程，如果是关闭超时，
// 可调用PendingTaskCount()获取处理中的任务数（这些任务正由线程池协程处理，导致阻塞关闭）
func (b *BasePool) CloseWithTimeout(t time.Duration) {
	if !b.closed.CASwap(false) {
		return
	}

	close(b.quit)
	b.drainPendingTasks()
	b.waitAllWorkerExit(t)

	close(b.permit)
	close(b.taskQueue)
	close(b.allWorkerExitSign)
}

//消耗所有任务队列里待完成的任务
func (b *BasePool) drainPendingTasks() []func() {
	pendingTasks := make([]func(), 0)
	for {
		select {
		case task := <-b.taskQueue:
			pendingTasks = append(pendingTasks, task)
			b.onTaskDone()
		default:
			return pendingTasks
		}
	}
}

//等待所有协程退出
func (b *BasePool) waitAllWorkerExit(timeout time.Duration) {
	if b.workerCount.Value() == 0 {
		return
	}

	if timeout < 0 {
		<-b.allWorkerExitSign
		return
	}

	//如果任务是无限循环或sleep()较长时间，则以下依靠超时退出等待
	//但后续关闭b.permit,b.taskQueue后再写入这些关闭的管道将报错
	DoTimeLimitTask(timeout, func() {
		<-b.allWorkerExitSign
	})

	//如果是等待超时，则还可能存在进行中的任务(如无限循环或长期阻塞的任务)
	//以下代码将使pool.Wait()不再阻塞等待的协程
	if b.pendingTaskCount.Value() > 0 &&
		b.allTaskDone.CASwap(false) {
		b.allTaskDoneWg.Done()
	}
}

//等待协程池里当前所有任务执行完成，可以多次调用等待
//注意：此方法仅支持添加任务的协程调用，不支持多协程调用
//如果要支持多协程调用，将涉及使用锁或修改Submit()添加任务逻辑，严重降低性能，暂不考虑
func (b *BasePool) Wait() {
	if b.pendingTaskCount.Value() == 0 {
		return
	}

	//如果wg.Wait()->wg.Add()将报错，所以使用AllTaskDone做二次判断
	if b.allTaskDone.False() {
		//time.Sleep(10*time.Nanosecond)
		b.allTaskDoneWg.Wait()
	}
}

//提交任务到协程池。不建议直接调用此方法，而使用RunXX()添加任务
func (b *BasePool) Submit(task func()) {
	if b.closed.True() || task == nil {
		return
	}

	select {
	case <-b.quit:
		return
	case b.taskQueue <- task:
		b.onTaskAdd()
	case b.permit <- struct{}{}:
		b.startWorker()
		b.taskQueue <- task
		b.onTaskAdd()
	}
}

//执行任务
func (b *BasePool) Run(task func()) <-chan struct{} {
	return run(b, task)
}

//执行可取消任务
func (b *BasePool) RunCancelable(ctx context.Context, task func()) <-chan struct{} {
	return runCancelable(ctx, b, task)
}

//执行可超时任务
func (b *BasePool) RunTimeLimit(timeout time.Duration, task func()) <-chan struct{} {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	return runCancelable(ctx, b, func() {
		defer cancel()
		task()
	})
}

//执行任务
func (b *BasePool) RunTask(task Task) <-chan Result {
	return runTask(b, task)
}

//执可取消行任务
func (b *BasePool) RunCancelableTask(ctx context.Context, task Task) <-chan Result {
	return runCancelableTask(b, ctx, task)
}

//执行可超时任务
func (b *BasePool) RunTimeLimitTask(timeout time.Duration, task Task) <-chan Result {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	return b.RunCancelableTask(ctx, ToTask(func() Result {
		defer time.AfterFunc(1*time.Nanosecond, cancel)
		return task.Run()
	}))
}

//执行可取消定时任务
func (b *BasePool) RunCancelableInterval(ctx context.Context, interval time.Duration, task func(ctx Context)) <-chan struct{} {
	return runCancelableInterval(b, ctx, interval, interval, task)
}

//执行可超时定时任务
func (b *BasePool) RunTimeLimitInterval(interval, timeout time.Duration, task func(ctx Context)) <-chan struct{} {
	c, cancel := context.WithTimeout(context.Background(), timeout)
	return b.RunCancelableInterval(c, interval, func(ctx Context) {
		task(ctx)

		if ctx.Done() {
			cancel()
		}
	})
}

//执行任务
func (b *BasePool) execTask(task func()) {
	//taskQueue关闭后，导致获取的task可能为nil
	if task == nil {
		return
	}

	defer b.onTaskDone()
	task() //可考虑处理panic
}

func (b *BasePool) onTaskAdd() {
	if b.pendingTaskCount.Incr(1) == 1 &&
		b.allTaskDone.CASwap(true) {
		b.allTaskDoneWg.Add(1)
	}
}

func (b *BasePool) onTaskDone() {
	if b.pendingTaskCount.Incr(-1) == 0 &&
		b.allTaskDone.CASwap(false) {

		//会有很小的概率发生以下错误:
		//协程1先执行onTaskAdd(), 但还未调用b.allTaskDoneWg.Add(1)
		//协程2后执行onTaskDone(),但先调用b.allTaskDoneWg.Done()
		//当任务执行时间很短，比如1个空函数，并且多次调用pool.Wait(),就可能出现此情况
		//
		//为了解决此问题，这里让协程2沉睡10纳秒，即让协程1先执行完onTaskAdd()，协程2再执行
		//经测试，即使设置为1纳秒，启动1000个协程，执行10000个空函数任务，循环等待2000次，不会报错
		//
		//也可以让onTaskAdd()/onTaskDone()进入时就获取锁，但这样做严重降低性能。以下解决方法是一种妥协
		time.Sleep(10 * time.Nanosecond)
		b.allTaskDoneWg.Done()
	}
}

func (b *BasePool) onWorkerExit() {
	<-b.permit //释放许可

	if b.workerCount.Incr(-1) == 0 {
		//忽略写入已关闭管道的错误
		defer util.OnExit(func(err error) {})
		b.allWorkerExitSign <- struct{}{}
	}
}

//创建协程
func (b *BasePool) startWorker() {
	n := b.workerCount.Incr(1)

	//启动核心worker
	if n <= b.minWorkerSize {
		b.startCoreWorker()
		return
	}

	//启动辅助worker，超时后退出
	b.startAssistWorker()
}

//创建核心协程，这些协程将一直保留在协程池里直到协程池关闭
func (b *BasePool) startCoreWorker() {
	go func() {
		defer b.onWorkerExit()

		for {
			select {
			case <-b.quit:
				return
			case task := <-b.taskQueue:
				b.execTask(task)
			}
		}
	}()
}

//创建辅助协程，这些协程将在空闲一段时间后退出
func (b *BasePool) startAssistWorker() {
	go func() {
		defer b.onWorkerExit()

		timeout := 10 * time.Second
		timer := time.NewTimer(timeout)

		for {
			select {
			case <-b.quit:
				timer.Stop()
				return
			case <-timer.C:
				return
			case task := <-b.taskQueue:
				b.execTask(task)
				timer.Reset(timeout)
			}
		}
	}()
}
