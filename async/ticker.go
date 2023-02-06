package async

import (
	"github.com/bingooh/b-go-util/util"
	"sync"
	"sync/atomic"
	"time"
)

type TickerOption struct {
	Period   time.Duration //tick触发周期
	MinDelay time.Duration //距离上次tick事件消费时间的最小等待时长，超过后才可以触发tick。默认为Period
	MinCount int64         //CurrentCount >= MinCount可以触发tick
	MaxCount int64         //CurrentCount >= MaxCount立刻触发tick
}

func NewTickerOption(minCount, maxCount int64, period time.Duration) TickerOption {
	return TickerOption{Period: period, MinCount: minCount, MaxCount: maxCount}
}

func (o TickerOption) MustNormalize() TickerOption {
	util.AssertOk(o.Period >= 0, `period<0`)
	util.AssertOk(o.MinCount >= 0, `minCount<0`)
	util.AssertOk(o.MaxCount >= 0, `maxCount<0`)
	util.AssertOk(o.MinCount <= o.MaxCount, `minCount[%v]>maxCount[%v]`, o.MinCount, o.MaxCount)
	util.AssertOk(o.MinCount > 0 || o.MaxCount > 0 || o.Period > 0, `minCount,maxCount,period不能都为0`)

	//time.Ticker触发的事件大概有几十毫秒误差，以下减去误差
	if o.Period > 0 && o.MinDelay <= 0 {
		if o.Period < 100*time.Millisecond {
			o.MinDelay = o.Period
		} else {
			o.MinDelay = o.Period - 100*time.Millisecond
		}
	}

	return o
}

// Ticker 定时器
// 满足以下条件会触发tick：
// - currentCount>=maxCount
// - ticker fired && currentCount>=minCount && time.Since(lastTickTime)>=minDelay
type Ticker struct {
	option       TickerOption
	ticker       *time.Ticker
	tickStream   chan time.Time
	C            <-chan time.Time
	isClosed     *util.AtomicBool
	lastTickTime atomic.Value //time.Time
	currentCount *util.AtomicInt64
}

func NewTicker(option TickerOption) *Ticker {
	t := &Ticker{
		option:       option.MustNormalize(),
		tickStream:   make(chan time.Time, 1),
		isClosed:     util.NewAtomicBool(false),
		currentCount: util.NewAtomicInt64(0),
	}
	t.C = t.tickStream
	t.lastTickTime.Store(time.Now())

	t.initTimeTicker()
	return t
}

func (t *Ticker) Close() {
	if !t.isClosed.CASwap(false) {
		return
	}

	//如果t.currentCount>0，考虑是否最后触发1次tick
	if t.ticker != nil {
		t.ticker.Stop()
	}

	close(t.tickStream)
}

func (t *Ticker) initTimeTicker() {
	if t.option.Period <= 0 {
		return
	}

	t.ticker = time.NewTicker(t.option.Period)
	go func() {
		for _ = range t.ticker.C {
			t.handleTickEvent()
		}
	}()
}

func (t *Ticker) handleTickEvent() {
	if t.option.MinCount > 0 && t.currentCount.Value() < t.option.MinCount {
		return
	}

	if t.option.MinDelay > 0 {
		last := t.lastTickTime.Load().(time.Time)
		if time.Since(last) < t.option.MinDelay {
			return
		}
	}

	t.fireTick()
}

func (t *Ticker) Count() int {
	if t.isClosed.False() {
		return t.currentCount.Int()
	}

	return 0
}

func (t *Ticker) ResetCount() {
	if t.isClosed.False() {
		t.currentCount.Set(0)
	}
}

func (t *Ticker) IncrCount(n int) {
	if n > 0 && t.option.MaxCount > 0 && t.isClosed.False() &&
		t.currentCount.Incr(int64(n)) >= t.option.MaxCount {
		t.fireTick()
	}
}

func (t *Ticker) fireTick() bool {
	if t.isClosed.True() {
		return false
	}
	defer util.Recover() //可能写入已关闭管道

	now := time.Now()
	cc := t.currentCount.Value()

	select {
	case t.tickStream <- now:
		t.currentCount.Incr(-cc)
		t.lastTickTime.Store(now)
		return true
	default:
		return false
	}
}

type TickerExecutor struct {
	lock     sync.RWMutex
	ticker   *Ticker
	isClosed bool
	tasks    []interface{}
	handler  func(tasks []interface{})
}

func NewTickerExecutor(option TickerOption, handler func(tasks []interface{})) *TickerExecutor {
	util.AssertOk(handler != nil, `handler为空`)
	e := &TickerExecutor{
		handler: handler,
		ticker:  NewTicker(option.MustNormalize()),
	}

	go func() {
		for _ = range e.ticker.C {
			e.InvokeNow()
		}
	}()

	return e
}

func (e *TickerExecutor) TaskSize() int {
	e.lock.RLock()
	defer e.lock.RUnlock()

	return len(e.tasks)
}

func (e *TickerExecutor) Close() {
	e.lock.Lock()
	defer e.lock.Unlock()

	if !e.isClosed {
		e.isClosed = true
		e.ticker.Close()
		e.invoke()
	}
}

func (e *TickerExecutor) Add(count int, tasks ...interface{}) {
	e.lock.Lock()
	defer e.lock.Unlock()

	if !e.isClosed {
		e.tasks = append(e.tasks, tasks...)
		e.ticker.IncrCount(count)
	}
}

func (e *TickerExecutor) InvokeNow() {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.invoke()
}

func (e *TickerExecutor) invoke() int {
	if len(e.tasks) == 0 {
		return 0
	}

	tasks := e.tasks
	e.tasks = nil
	e.ticker.ResetCount()
	e.handler(tasks)
	return len(tasks)
}
