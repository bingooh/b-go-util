package async

import (
	"github.com/bingooh/b-go-util/util"
	"sync"
	"time"
)

type Barrier struct {
	lock sync.Locker
}

func NewBarrier() *Barrier {
	return NewBarrierWithLocker(&sync.Mutex{})
}

func NewBarrierWithLocker(lock sync.Locker) *Barrier {
	util.AssertOk(lock != nil, `lock为空`)
	return &Barrier{lock: lock}
}

func (b *Barrier) Do(fn func()) {
	b.lock.Lock()
	defer b.lock.Unlock()

	fn()
}

func (b *Barrier) Invoke(fn func() error) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	return fn()
}

// PeriodBarrier 每个周期最多执行1次函数
type PeriodBarrier struct {
	period         time.Duration
	lastInvokeTime *util.AtomicTime
	isInvoking     *util.AtomicBool
}

func NewPeriodBarrier(period time.Duration) *PeriodBarrier {
	util.AssertOk(period > 0, `invalid period[%v]`, period)

	return &PeriodBarrier{
		period:         period,
		lastInvokeTime: util.NewAtomicTime(),
		isInvoking:     util.NewAtomicBool(false),
	}
}

func (b *PeriodBarrier) allowInvoke() bool {
	if time.Since(b.lastInvokeTime.Value()) < b.period {
		return false
	}

	return b.isInvoking.CASwap(false)
}

func (b *PeriodBarrier) afterInvoke() {
	b.lastInvokeTime.Set(time.Now())
	b.isInvoking.Set(false)
}

func (b *PeriodBarrier) Do(fn func()) {
	if b.allowInvoke() {
		defer b.afterInvoke()
		fn()
	}
}

func (b *PeriodBarrier) Invoke(fn func() error) error {
	if b.allowInvoke() {
		defer b.afterInvoke()
		return fn()
	}

	return nil
}
