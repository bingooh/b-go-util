package async

import (
	"github.com/bingooh/b-go-util/util"
	"time"
)

type timerExecutorTask struct {
	delay     time.Duration //等待时长
	key       interface{}   //任务key
	value     interface{}   //任务value
	bucketIdx int           //任务所在bucket索引值
	round     int           //任务所在time wheel层数
	removed   bool          //是否已移除
	cyclic    bool          //是否循环执行
}

// TimerExecutor 超时事件执行器，内部使用TimeWheel触发timeout事件
// 基本流程：每次tick事件，处理1个bucket里的全部timer
type TimerExecutor struct {
	tasks            map[interface{}]*timerExecutorTask   //key:task key
	buckets          []map[interface{}]*timerExecutorTask //key:task key
	bucketsNum       int                                  //bucket总数
	currentBucketIdx int                                  //当前bucket索引值
	ticker           *time.Ticker
	period           time.Duration
	handler          func(key, value interface{})
	onClosedHandler  func(key, value interface{})

	putCh       chan *timerExecutorTask
	delCh       chan interface{}
	runDoneCh   *DoneChannel
	closeDoneCh *DoneChannel
}

// NewTimerExecutor 创建执行器
// period tick触发周期
// bucketsNum 保存定时任务bucket数量
// 如果period=1s,bucketsNum=60，则遍历1次全部bucket需要60s
// 每次tick启动1个后台协程执行当前bucket里的任务，handler需自行决定是否启用多协程加快任务处理速度
// 执行器实现time wheel以提高定时器性能，但不能保证定时精度。如period=10s，任务延迟1秒执行，则任务可能在0-10s内任意时刻执行
func NewTimerExecutor(period time.Duration, bucketsNum int, handler func(key, value interface{})) *TimerExecutor {
	util.AssertOk(period > 0, `period<=0`)
	util.AssertOk(bucketsNum > 0, `bucketNum<=0`)
	util.AssertOk(handler != nil, `handler is nil`)

	e := &TimerExecutor{
		tasks:       make(map[interface{}]*timerExecutorTask),
		buckets:     make([]map[interface{}]*timerExecutorTask, bucketsNum),
		bucketsNum:  bucketsNum,
		ticker:      time.NewTicker(period),
		period:      period,
		handler:     handler,
		putCh:       make(chan *timerExecutorTask),
		delCh:       make(chan interface{}),
		runDoneCh:   NewDoneChannel(),
		closeDoneCh: NewDoneChannel(),
	}

	for i := 0; i < bucketsNum; i++ {
		e.buckets[i] = make(map[interface{}]*timerExecutorTask)
	}

	go e.run()
	return e
}

// WithOnClosedHandler 关闭后回调函数，处理关闭后还未处理的待执行的任务
func (e *TimerExecutor) WithOnClosedHandler(fn func(key, value interface{})) *TimerExecutor {
	e.onClosedHandler = fn
	return e
}

func (e *TimerExecutor) Close() {
	if e.closeDoneCh.Close() {
		<-e.runDoneCh.Done() //等待onClosedHandler()执行完成
	}
}

func (e *TimerExecutor) Put(delay time.Duration, key interface{}) bool {
	return e.PutTask(delay, key, nil, false)
}

func (e *TimerExecutor) PutTask(delay time.Duration, key, value interface{}, cyclic bool) bool {
	util.AssertOk(delay > 0, `delay<=0`)
	util.AssertOk(key != nil, `key is nil`)

	task := &timerExecutorTask{
		delay:  delay,
		key:    key,
		value:  value,
		cyclic: cyclic,
	}

	select {
	case e.putCh <- task:
		return true
	case <-e.closeDoneCh.Done():
		return false
	}
}

func (e *TimerExecutor) Del(key interface{}) bool {
	util.AssertOk(key != nil, `key is nil`)

	select {
	case e.delCh <- key:
		return true
	case <-e.closeDoneCh.Done():
		return false
	}
}

func (e *TimerExecutor) run() {
	for {
		select {
		case <-e.ticker.C:
			e.onTick()
		case task := <-e.putCh:
			e.onPut(task)
		case key := <-e.delCh:
			if task, ok := e.tasks[key]; ok {
				task.removed = true
			}
		case <-e.closeDoneCh.Done():
			e.onClosed()
			return
		}
	}
}

func (e *TimerExecutor) onClosed() {
	defer e.runDoneCh.Close()

	e.ticker.Stop()

	if e.onClosedHandler != nil {
		for key, tm := range e.tasks {
			e.onClosedHandler(key, tm.value)
		}
	}
}

func (e *TimerExecutor) onPut(task *timerExecutorTask) {
	if old, ok := e.tasks[task.key]; ok {
		old.removed = true
	}

	//如果新旧task.bucketIdx相同，旧task将被覆盖
	e.putTask(task)
}

func (e *TimerExecutor) onTick() {
	defer func() {
		e.currentBucketIdx++
		if e.currentBucketIdx == e.bucketsNum {
			e.currentBucketIdx = 0
		}
	}()

	//执行当前bucket包含的任务
	bucket := e.buckets[e.currentBucketIdx]

	//fmt.Println(`tick:`,e.currentBucketIdx,len(bucket))//todo del it
	if len(bucket) == 0 {
		return
	}

	var tasks []*timerExecutorTask
	for _, task := range bucket {
		switch {
		case task.removed:
			e.delTask(task)
		case task.round > 0:
			task.round--
		default:
			e.delTask(task)
			tasks = append(tasks, task)
		}
	}

	if len(tasks) > 0 {
		go e.runTasks(tasks) //避免阻塞主协程
	}
}

func (e *TimerExecutor) resetTaskPosition(task *timerExecutorTask) {
	steps := int(task.delay / e.period)
	task.bucketIdx = (steps + e.currentBucketIdx) % e.bucketsNum
	task.round = steps / e.bucketsNum
}

func (e *TimerExecutor) putTask(task *timerExecutorTask) {
	e.resetTaskPosition(task)
	e.tasks[task.key] = task
	e.buckets[task.bucketIdx][task.key] = task
}

func (e *TimerExecutor) delTask(task *timerExecutorTask) {
	delete(e.tasks, task.key)
	delete(e.buckets[task.bucketIdx], task.key)
}

func (e *TimerExecutor) runTasks(tasks []*timerExecutorTask) {
	for _, task := range tasks {
		e.handler(task.key, task.value)

		//每次tick启动1个新协程执行runTasks，如果循环任务延迟时间很短，
		//可能会有多个协程添加循环任务，以下方法可能阻塞导致后续任务执行延时
		if task.cyclic {
			e.PutTask(task.delay, task.key, task.value, true)
		}
	}
}
