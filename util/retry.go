package util

import (
	"sync"
	"time"
)

//重试计数器
type RetryCounter interface {
	Count() int                  //当前重试次数
	NextInterval() time.Duration //下1个间隔时长，如果已达最大重试次数，则返回0
}

//简单重试计数器
type retryCounter struct {
	lock         sync.Mutex
	count        int           //当前重试次数
	maxCount     int           //最大重试次数,0表示不限制
	interval     time.Duration //当前间隔时长
	maxInterval  time.Duration //最大间隔时长，0表示不限制
	stepInterval time.Duration //每次重试增加的间隔时长
}

//固定间隔时长重试计数器
func NewRetryCounter(maxCount int, interval time.Duration) RetryCounter {
	return NewStepRetryCounter(maxCount, interval, 0, 0)
}

//步进间隔时长重试计数器
func NewStepRetryCounter(maxCount int, initInterval, stepInterval, maxInterval time.Duration) RetryCounter {
	AssertOk(maxCount >= 0, `maxCount小于0`)
	AssertOk(initInterval >= 0, `initInterval小于0`)
	AssertOk(stepInterval >= 0, `stepInterval小于0`)
	AssertOk(maxInterval >= 0, `maxInterval小于0`)
	AssertOk(initInterval > 0 || stepInterval > 0, `initInterval和stepInterval同时等于0`)

	return &retryCounter{
		interval:     initInterval,
		maxCount:     maxCount,
		maxInterval:  maxInterval,
		stepInterval: stepInterval,
	}
}

func (r *retryCounter) Count() int {
	return r.count
}

func (r *retryCounter) NextInterval() time.Duration {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.maxCount > 0 && r.count >= r.maxCount {
		return 0
	}

	r.count++

	if r.stepInterval > 0 && (r.maxInterval <= 0 || r.interval < r.maxInterval) {
		r.interval += r.stepInterval

		if r.maxInterval > 0 && r.interval > r.maxInterval {
			r.interval = r.maxInterval
		}
	}

	return r.interval
}

//参数fn为要重试执行的任务，如果返回nil表示执行成功
//fn会被立刻执行1次，如果失败则等待下次重试
func DoRetry(counter RetryCounter, fn func() error) error {
	AssertOk(counter != nil, `counter为空`)
	AssertOk(fn != nil, `fn为空`)

	for {
		err := fn()
		if err == nil {
			return nil
		}

		sleep := counter.NextInterval()
		if sleep <= 0 {
			return err
		}

		time.Sleep(sleep)
	}
}
