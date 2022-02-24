package util

import (
	"sync/atomic"
)

type AtomicBool struct {
	value *int32
}

func NewAtomicBool(v bool) *AtomicBool {
	var i int32 = 0
	if v {
		i = 1
	}

	return &AtomicBool{value: &i}
}

func (b *AtomicBool) Set(v bool) {
	var i int32 = 0
	if v {
		i = 1
	}
	atomic.StoreInt32(b.value, i)
}

//compare and swap
func (b *AtomicBool) CASwap(expect bool) bool {
	var old int32 = 0
	var new int32 = 1
	if expect {
		old = 1
		new = 0
	}

	return atomic.CompareAndSwapInt32(b.value, old, new)
}

func (b *AtomicBool) Value() bool {
	return atomic.LoadInt32(b.value) == 1
}

func (b *AtomicBool) True() bool {
	return b.Value() == true
}

func (b *AtomicBool) False() bool {
	return b.Value() == false
}
