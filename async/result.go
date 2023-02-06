package async

import (
	"context"
	"errors"
	"github.com/bingooh/b-go-util/util"
	"sync"
)

var TypeCastErr = errors.New(`type cast err`)

// Result 任务执行结果
// Value()返回任务执行结果的值，同时提供Bool()/Int()等帮助方法将结果值转换为对应的数据类型
// 目前实现仅使用简单的数据类型转换，如Int()基本等价于Value().(int)，但转换失败返回util.TypeCastErr
type Result interface {
	Error() error       //任务返回的错误
	Canceled() bool     //任务是否取消
	Timeout() bool      //任务是否超时
	Value() interface{} //任务返回的值
	Bool() (bool, error)
	Int() (int, error)
	Int32() (int32, error)
	Int64() (int64, error)
	String() (string, error)
	MustBool() bool
	MustInt() int
	MustInt32() int32
	MustInt64() int64
	MustString() string
}

// BaseResult Result实现类
type BaseResult struct {
	value    interface{}
	err      error
	canceled bool
	timeout  bool
}

func NewResult(v interface{}, err error) Result {
	return &BaseResult{value: v, err: err}
}

func NewResultWithCtx(ctx Context) Result {
	return &BaseResult{canceled: ctx.Canceled(), timeout: ctx.Timeout(), err: ctx.Error()}
}

func NewResultWithContext(ctx context.Context) Result {
	return NewResultWithCtx(NewContext(ctx))
}

func (b *BaseResult) Error() error {
	return b.err
}

func (b *BaseResult) Canceled() bool {
	return b.canceled
}

func (b *BaseResult) Timeout() bool {
	return b.timeout
}

func (b *BaseResult) Value() interface{} {
	return b.value
}

func (b *BaseResult) Bool() (bool, error) {
	if b.err != nil {
		return false, b.err
	}

	if v, ok := b.value.(bool); ok {
		return v, nil
	}

	return false, TypeCastErr
}

func (b *BaseResult) Int() (int, error) {
	if b.err != nil {
		return 0, b.err
	}

	if v, ok := b.value.(int); ok {
		return v, nil
	}

	return 0, TypeCastErr
}

func (b *BaseResult) Int32() (int32, error) {
	if b.err != nil {
		return 0, b.err
	}

	if v, ok := b.value.(int32); ok {
		return v, nil
	}

	return 0, TypeCastErr
}

func (b *BaseResult) Int64() (int64, error) {
	if b.err != nil {
		return 0, b.err
	}
	if v, ok := b.value.(int64); ok {
		return v, nil
	}

	return 0, TypeCastErr
}

func (b *BaseResult) String() (string, error) {
	if b.err != nil {
		return "", b.err
	}

	if v, ok := b.value.(string); ok {
		return v, nil
	}

	return "", TypeCastErr
}

func (b *BaseResult) MustBool() bool {
	v, err := b.Bool()
	b.assertNilErr(err)
	return v
}

func (b *BaseResult) MustInt() int {
	v, err := b.Int()
	b.assertNilErr(err)
	return v
}

func (b *BaseResult) MustInt32() int32 {
	v, err := b.Int32()
	b.assertNilErr(err)
	return v
}

func (b *BaseResult) MustInt64() int64 {
	v, err := b.Int64()
	b.assertNilErr(err)
	return v
}

func (b *BaseResult) MustString() string {
	v, err := b.String()
	b.assertNilErr(err)
	return v
}

func (b *BaseResult) assertNilErr(err error) {
	util.AssertNilErr(err)
}

type ResultList struct {
	values []Result
}

func NewResultList() *ResultList {
	return &ResultList{values: make([]Result, 0)}
}

func (l *ResultList) Size() int {
	return len(l.values)
}

func (l *ResultList) Has(idx int) bool {
	return idx >= 0 && idx < len(l.values)
}

func (l *ResultList) Get(idx int) Result {
	if idx >= 0 && idx < len(l.values) {
		return l.values[idx]
	}

	return nil
}

func (l *ResultList) Add(results ...Result) {
	for _, result := range results {
		l.values = append(l.values, result)
	}
}

func (l *ResultList) ToSlice() []Result {
	rs := make([]Result, len(l.values))
	copy(rs, l.values)

	return rs
}

type SyncResultList struct {
	lock   sync.RWMutex
	values []Result
}

func NewSyncResultList() *SyncResultList {
	return &SyncResultList{values: make([]Result, 0)}
}

func (l *SyncResultList) Size() int {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return len(l.values)
}

func (l *SyncResultList) Has(idx int) bool {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return idx >= 0 && idx < len(l.values)
}

func (l *SyncResultList) Get(idx int) Result {
	l.lock.RLock()
	defer l.lock.RUnlock()

	if idx >= 0 && idx < len(l.values) {
		return l.values[idx]
	}

	return nil
}

func (l *SyncResultList) Add(results ...Result) {
	l.lock.Lock()
	defer l.lock.Unlock()

	for _, result := range results {
		l.values = append(l.values, result)
	}
}

func (l *SyncResultList) ToSlice() []Result {
	l.lock.RLock()
	defer l.lock.RUnlock()

	rs := make([]Result, len(l.values))
	copy(rs, l.values)

	return rs
}

type ResultMap struct {
	values map[int]Result
}

func NewResultMap() *ResultMap {
	return &ResultMap{values: make(map[int]Result, 0)}
}

func (m *ResultMap) Size() int {
	return len(m.values)
}

func (m *ResultMap) Has(key int) bool {
	_, ok := m.values[key]
	return ok
}

func (m *ResultMap) Get(key int) Result {
	return m.values[key]
}

func (m *ResultMap) Put(key int, val Result) {
	m.values[key] = val
}

func (m *ResultMap) Del(key int) Result {
	if old, ok := m.values[key]; ok {
		delete(m.values, key)
		return old
	}

	return nil
}

func (m *ResultMap) ToMap() map[int]Result {
	rs := make(map[int]Result, len(m.values))

	for k, v := range m.values {
		rs[k] = v
	}

	return rs
}

type SyncResultMap struct {
	lock   sync.RWMutex
	values map[int]Result
}

func NewSyncResultMap() *SyncResultMap {
	return &SyncResultMap{values: make(map[int]Result, 0)}
}

func (m *SyncResultMap) Size() int {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return len(m.values)
}

func (m *SyncResultMap) Has(key int) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()

	_, ok := m.values[key]
	return ok
}

func (m *SyncResultMap) Get(key int) Result {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.values[key]
}

func (m *SyncResultMap) Put(key int, val Result) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.values[key] = val
}

func (m *SyncResultMap) Del(key int) Result {
	m.lock.Lock()
	defer m.lock.Unlock()

	if old, ok := m.values[key]; ok {
		delete(m.values, key)
		return old
	}

	return nil
}

func (m *SyncResultMap) ToMap() map[int]Result {
	m.lock.RLock()
	defer m.lock.RUnlock()

	rs := make(map[int]Result, len(m.values))

	for k, v := range m.values {
		rs[k] = v
	}

	return rs
}
