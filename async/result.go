package async

import (
	"context"
	"errors"
	"github.com/bingooh/b-go-util/util"
	"sync"
)

var TypeCastErr = errors.New(`type cast err`)

// 任务执行结果
// Value()返回任务执行结果的值，同时提供Bool()/Int()等帮助方法将结果值转换为对应的数据类型
// 目前实现仅使用简单的数据类型转换，如Int()基本等价于Value().(int)，但转换失败返回util.TypeCastErr
type Result interface {
	Error() error       //任务返回的错误
	HasError() bool     //任务是否有错
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

type ResultList []Result
type ResultMap map[int]Result //key为结果索引值

//遍历结果
func (l ResultList) Each(fn func(i int, result Result)) {
	for i, r := range l {
		fn(i, r)
	}
}

//遍历结果，直到fn返回false
func (l ResultList) ForEach(fn func(i int, result Result) bool) {
	for i, r := range l {
		if !fn(i, r) {
			return
		}
	}
}

//遍历结果
func (m ResultMap) Each(fn func(key int, result Result)) {
	for k, v := range m {
		fn(k, v)
	}
}

//遍历结果，直到fn返回false
func (m ResultMap) ForEach(fn func(key int, result Result) bool) {
	for k, v := range m {
		if !fn(k, v) {
			return
		}
	}
}

//Result实现类
type BaseResult struct {
	lock     sync.Mutex
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
	return NewResultWithCtx(NewCtx(ctx))
}

func (b *BaseResult) Error() error {
	return b.err
}

func (b *BaseResult) HasError() bool {
	return b.err != nil
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
