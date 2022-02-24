package util

import "golang.org/x/net/context"

type CancelableContext struct {
	ctx    context.Context
	cancel context.CancelFunc
}

func NewCancelableContext() *CancelableContext {
	return NewCancelableContextWithParent(context.Background())
}

func NewCancelableContextWithParent(parent context.Context) *CancelableContext {
	ctx, cancel := context.WithCancel(parent)
	return &CancelableContext{ctx: ctx, cancel: cancel}
}

func (c *CancelableContext) Context() context.Context {
	return c.ctx
}

func (c *CancelableContext) Cancel() {
	c.cancel()
}
