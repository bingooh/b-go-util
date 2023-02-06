package util

import (
	"strconv"
	"sync/atomic"
	"time"
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

type AtomicTime struct {
	value atomic.Value
}

func NewAtomicTime() *AtomicTime {
	return &AtomicTime{}
}

func (a *AtomicTime) Set(v time.Time) *AtomicTime {
	a.value.Store(v)
	return a
}

func (a *AtomicTime) Value() time.Time {
	if v, ok := a.value.Load().(time.Time); ok {
		return v
	}

	return time.Time{}
}

func (a *AtomicTime) CASwap(expect, target time.Time) bool {
	return a.value.CompareAndSwap(expect, target)
}

type packedError struct{ Value error }

func packError(v error) interface{} {
	return packedError{v}
}

func unpackError(v interface{}) error {
	if err, ok := v.(packedError); ok {
		return err.Value
	}
	return nil
}

type AtomicError struct {
	value atomic.Value
}

func NewAtomicError() *AtomicError {
	a := &AtomicError{}
	return a.Set(nil)
}

func (a *AtomicError) Value() error {
	return unpackError(a.value.Load())
}

func (a *AtomicError) Set(err error) *AtomicError {
	a.value.Store(packError(err))
	return a
}

func (a *AtomicError) SetIfAbsent(err error) bool {
	return a.CASwap(nil, err)
}

func (a *AtomicError) CASwap(expect, target error) bool {
	return a.value.CompareAndSwap(packError(expect), packError(target))
}
