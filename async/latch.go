package async

import (
	"b-go-util/util"
	"sync"
)

//门闩，调用latch.Wait()将阻塞当前线程，直到调用latch.Open()
//如果调用sync.WaitGroup：wait()->add()，可能会报崩溃错误，必须等上次等待的协程全部释放后才能再次调用add(),即可理解为wg不可重用
type Latch struct {
	isClosed *util.AtomicBool
	cond     *sync.Cond
}

//创建门闩
func NewLatch(isClosed bool) *Latch {
	l := &Latch{
		cond:     sync.NewCond(&sync.Mutex{}),
		isClosed: util.NewAtomicBool(isClosed),
	}

	return l
}

func (l *Latch) Open() {
	if l.isClosed.True() {
		l.CASwap(true)
	}
}

//打开门闩，如果门闩已经打开返回false,如果从关闭变为打开返回true
func (l *Latch) CAOpen() bool {
	if l.isClosed.False() {
		return false
	}

	return l.CASwap(true)
}

func (l *Latch) Close() {
	if l.isClosed.False() {
		l.CASwap(false)
	}
}

//关闭门闩，如果门闩已经关闭返回false,如果从打开变为关闭返回true
func (l *Latch) CAClose() bool {
	if l.isClosed.True() {
		return false
	}

	return l.CASwap(false)
}

//比较latch的关闭状态是否与expectIsClosed相同
//如果不同返回false，如果相同则返回true，且设置latch关闭状态为!expectIsClosed
func (l *Latch) CASwap(expectIsClosed bool) bool {
	l.cond.L.Lock()
	defer l.cond.L.Unlock()

	if !l.isClosed.CASwap(expectIsClosed) {
		return false
	}

	if l.isClosed.False() {
		//说明从关闭变为打开
		l.cond.Broadcast()
	}

	return true
}

func (l *Latch) IsClosed() bool {
	return l.isClosed.Value()
}

func (l *Latch) Wait() {
	if l.isClosed.False() {
		return
	}

	l.cond.L.Lock()

	for l.isClosed.True() {
		l.cond.Wait()
	}

	l.cond.L.Unlock()
}
