package async

import (
	"github.com/bingooh/b-go-util/util"
)

type DoneChannel struct {
	ch     chan struct{}
	isDone *util.AtomicBool
}

func NewDoneChannel() *DoneChannel {
	return &DoneChannel{
		ch:     make(chan struct{}),
		isDone: util.NewAtomicBool(false),
	}
}

func (c *DoneChannel) IsDone() bool {
	return c.isDone.Value()
}

func (c *DoneChannel) Done() <-chan struct{} {
	return c.ch
}

func (c *DoneChannel) Close() bool {
	if c.isDone.CASwap(false) {
		close(c.ch)
		return true
	}

	return false
}

type Doneable interface {
	Done() <-chan struct{}
}

func ToDoneable(v interface{}) (d Doneable, ok bool) {
	d, ok = v.(Doneable)
	return
}
