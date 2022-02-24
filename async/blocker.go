package async

import "sync"

//暂未使用：WaitGroup+ValueHolder
type Blocker struct {
	wg   sync.WaitGroup
	lock sync.RWMutex

	isBlockEnabled bool
	val            interface{}
}

func NewBlocker() *Blocker {
	h := &Blocker{isBlockEnabled: true}
	h.wg.Add(1)
	return h
}

func NewBlockerOf(val interface{}) *Blocker {
	return &Blocker{val: val}
}

//是否阻塞已启用
//如果此方法返回true，调用Get()将被阻塞，即说明Blocker未设置值
//注意：如果b.PutDetail(nil)，仍认为是已设置值，此时调用此方法将返回false
func (b *Blocker) IsBlockEnabled() bool {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.isBlockEnabled
}

func (b *Blocker) IsNilVal() bool {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.val == nil
}

//阻塞直到获取值
func (b *Blocker) Get() interface{} {
	b.wg.Wait()
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.val
}

//放入1个值将解除阻塞
func (b *Blocker) Put(val interface{}) {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.val = val

	if b.isBlockEnabled {
		b.wg.Done()
		b.isBlockEnabled = false
	}
}

//删除保存的值将启用阻塞
func (b *Blocker) Remove() interface{} {
	b.lock.Lock()
	defer b.lock.Unlock()

	val := b.val
	b.val = nil

	if !b.isBlockEnabled {
		b.wg.Add(1)
		b.isBlockEnabled = true
	}

	return val
}
