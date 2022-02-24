package util

import (
	"strconv"
	"sync/atomic"
)

type AtomicInt64 struct {
	value *int64
}

func NewAtomicInt64(v int64) *AtomicInt64 {
	return &AtomicInt64{&v}
}

func (s *AtomicInt64) Set(v int64) {
	atomic.StoreInt64(s.value, v)
}

func (s *AtomicInt64) Incr(n int64) int64 {
	return atomic.AddInt64(s.value, n)
}

func (s *AtomicInt64) CASwap(expect, target int64) bool {
	return atomic.CompareAndSwapInt64(s.value, expect, target)
}

func (s *AtomicInt64) Value() int64 {
	return atomic.LoadInt64(s.value)
}

func (s *AtomicInt64) Int() int {
	return int(s.Value())
}

func (s *AtomicInt64) Int32() int32 {
	return int32(s.Value())
}

func (s *AtomicInt64) String() string {
	return strconv.FormatInt(s.Value(), 10)
}
