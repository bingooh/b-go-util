package async

import "sync"

// Blocker 暂未使用：WaitGroup+ValueHolder
// todo 考虑改写为future，提供错误和正常返回
type Blocker struct {
	wg   sync.WaitGroup
	lock sync.RWMutex

	hasValue bool
	val      interface{}
}

func NewBlocker() *Blocker {
	h := &Blocker{}
	h.wg.Add(1)
	return h
}

func NewBlockerOf(val interface{}) *Blocker {
	return &Blocker{hasValue: true, val: val}
}

// HasValue 是否已设置值，如果有值则不会阻塞
// 如果此方法返回true，调用Get()将被阻塞，即说明Blocker未设置值
// 注意：如果b.Put(nil)，仍认为是已设置值，此时调用此方法将返回false
func (b *Blocker) HasValue() bool {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.hasValue
}

// Get 阻塞直到获取值
func (b *Blocker) Get() interface{} {
	b.wg.Wait()
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.val
}

// Peek 阻塞直到获取值
func (b *Blocker) Peek() interface{} {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.val
}

// Put 设置值将解除阻塞
func (b *Blocker) Put(val interface{}) {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.val = val

	if !b.hasValue {
		b.hasValue = true
		b.wg.Done()
	}
}

// Remove 删除保存的值将启用阻塞
func (b *Blocker) Remove() interface{} {
	b.lock.Lock()
	defer b.lock.Unlock()

	val := b.val
	b.val = nil

	if b.hasValue {
		b.hasValue = false
		b.wg.Add(1)
	}

	return val
}
