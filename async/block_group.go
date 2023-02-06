package async

import (
	"github.com/bingooh/b-go-util/_string"
	"github.com/bingooh/b-go-util/util"
	"sync"
)

// BlockGroup 阻塞组,可用于同步等待异步请求结果
type BlockGroup struct {
	lock    sync.RWMutex
	holders map[string]*blockResultHolder
}

type blockResultHolder struct {
	wg        sync.WaitGroup
	hasResult *util.AtomicBool
	result    interface{}
}

func newBlockResultHolder() *blockResultHolder {
	h := &blockResultHolder{
		hasResult: util.NewAtomicBool(false),
	}

	h.wg.Add(1)
	return h
}

func (h *blockResultHolder) Put(result interface{}) *blockResultHolder {
	h.result = result
	if h.hasResult.CASwap(false) {
		h.wg.Done()
	}

	return h
}

func (h *blockResultHolder) Unblock() {
	if h.hasResult.False() {
		h.wg.Done()
	}
}

func NewBlockGroup() *BlockGroup {
	return &BlockGroup{
		holders: make(map[string]*blockResultHolder),
	}
}

func (g *BlockGroup) Has(key string) bool {
	if _string.Empty(key) {
		return false
	}

	g.lock.RLock()
	defer g.lock.RUnlock()

	_, ok := g.holders[key]
	return ok
}

// Peek 获取key对应的值,不会阻塞
func (g *BlockGroup) Peek(key string) interface{} {
	if _string.Empty(key) {
		return nil
	}

	g.lock.RLock()
	defer g.lock.RUnlock()

	if holder, ok := g.holders[key]; ok {
		return holder.result
	}

	return nil
}

// Get 阻塞直到获取key对应的值
// 注意：调用此方法内部将创建1个holder对象，如不再使用需及时移除避免内存泄露
func (g *BlockGroup) Get(key string) interface{} {
	if _string.Empty(key) {
		return nil
	}

	g.lock.RLock()
	if holder, ok := g.holders[key]; ok {
		g.lock.RUnlock()
		holder.wg.Wait()
		return holder.result
	}
	g.lock.RUnlock()

	g.lock.Lock()
	holder, exist := g.holders[key]
	if !exist {
		holder = newBlockResultHolder()
		g.holders[key] = holder
	}
	g.lock.Unlock()

	holder.wg.Wait()
	return holder.result
}

// Put 设置key对应的值，将解除阻塞
func (g *BlockGroup) Put(key string, val interface{}) {
	util.AssertNotEmpty(key, `key为空`)
	g.lock.RLock()

	if holder, ok := g.holders[key]; ok {
		g.lock.RUnlock()
		holder.Put(val)
		return
	}
	g.lock.RUnlock()

	g.lock.Lock()
	if holder, exist := g.holders[key]; exist {
		holder.Put(val)
	} else {
		holder = newBlockResultHolder().Put(val)
		g.holders[key] = holder
	}
	g.lock.Unlock()
}

// PutIfExist 如果key存在则设置key对应的值并返回true，将解除阻塞
func (g *BlockGroup) PutIfExist(key string, val interface{}) bool {
	if _string.Empty(key) {
		return false
	}

	g.lock.RLock()
	defer g.lock.RUnlock()

	if holder, ok := g.holders[key]; ok {
		holder.Put(val)
		return true
	}

	return false
}

// Remove 删除key对应的值
func (g *BlockGroup) Remove(key string) interface{} {
	if _string.Empty(key) {
		return nil
	}

	g.lock.Lock()
	defer g.lock.Unlock()

	if holder, ok := g.holders[key]; ok {
		delete(g.holders, key)
		holder.Unblock() //避免协程阻塞在删除的holder
		return holder.result
	}

	return nil
}

func (g *BlockGroup) RemoveAll() {
	g.lock.Lock()
	defer g.lock.Unlock()

	if len(g.holders) == 0 {
		return
	}

	for _, holder := range g.holders {
		holder.Unblock()
	}

	g.holders = make(map[string]*blockResultHolder)
}
