package async

import (
	"b-go-util/util"
	"context"
)

//任务执行上下文
type Context interface {
	Done() bool     //任务是否完成
	Canceled() bool //任务是否取消
	Aborted() bool  //任务是否主动取消
	Timeout() bool  //任务是否超时
	Error() error   //任务返回的错误
	Abort()         //主动取消任务
	Count() int64   //任务循环执行次数，从1开始计数
}

type BaseCtx struct {
	ctx   context.Context
	abort *util.AtomicBool
	count *util.AtomicInt64 //执行次数,从1开始
}

// ctx可以为nil
func NewCtx(ctx context.Context) Context {
	return &BaseCtx{
		ctx:   ctx,
		abort: util.NewAtomicBool(false),
		count: util.NewAtomicInt64(0),
	}
}

func (c *BaseCtx) Done() bool {
	return c.Canceled() || c.Timeout() || c.abort.Value()
}

// 主动放弃，跳出当前循环
func (c *BaseCtx) Abort() {
	c.abort.CASwap(false)
}

func (c *BaseCtx) Aborted() bool {
	return c.abort.Value()
}

func (c *BaseCtx) incrCount() {
	c.count.Incr(1)
}

//循环执行次数,从1开始
func (c *BaseCtx) Count() int64 {
	return c.count.Value()
}

func (c *BaseCtx) Canceled() bool {
	return c.ctx != nil && c.ctx.Err() == context.Canceled
}

func (c *BaseCtx) Timeout() bool {
	return c.ctx != nil && c.ctx.Err() == context.DeadlineExceeded
}

func (c *BaseCtx) Error() error {
	if c.ctx != nil {
		return c.ctx.Err()
	}
	return nil
}

//内部使用，解决任务执行过程中context.Context完成导致任务的ctx.Done()执行2次问题
type FreezeCtx struct {
	ctx  Context
	done bool
}

//根据传入的done是否为true，来确定FreezeCtx是否超时或取消
// - done==true : FreezeCtx.Canceled()/Timeout()总是为false
// - done==false: FreezeCtx.Canceled()/Timeout()直接返回传入的ctx对应方法的值
func newFreezeCtx(ctx Context, done bool) *FreezeCtx {
	return &FreezeCtx{done: done, ctx: ctx}
}

func (c *FreezeCtx) Done() bool {
	return c.done || c.ctx.Aborted()
}

func (c *FreezeCtx) Abort() {
	c.ctx.Abort()
}

func (c *FreezeCtx) Aborted() bool {
	return c.ctx.Aborted()
}

func (c *FreezeCtx) Count() int64 {
	return c.ctx.Count()
}

func (c *FreezeCtx) Canceled() bool {
	if !c.Done() {
		return false
	}

	return c.ctx.Canceled()
}

func (c *FreezeCtx) Timeout() bool {
	if !c.Done() {
		return false
	}

	return c.ctx.Timeout()
}

func (c *FreezeCtx) Error() error {
	if !c.Done() {
		return nil
	}

	return c.ctx.Error()
}
