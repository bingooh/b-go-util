package async

import (
	"github.com/bingooh/b-go-util/_string"
	"github.com/bingooh/b-go-util/util"
	"sync"
)

//暂未使用：缓存任务组，执行任务获取其返回值作为缓存，下次获取直接返回缓存值
//使用single flight设计模式，避免击穿缓存。即仅在没有对应缓存时才执行第1个请求添加的任务去获取缓存值
type CacheGroup struct {
	lock    sync.RWMutex
	holders map[string]*resultHolder
}

type resultHolder struct {
	wg     sync.WaitGroup
	result Result
}

func NewCacheGroup() *CacheGroup {
	return &CacheGroup{
		holders: make(map[string]*resultHolder),
	}
}

//获取key对应的值，如果key不存在，则执行task获取值并保存到Group
func (g *CacheGroup) Do(key string, task Task) Result {
	util.AssertOk(!_string.Empty(key), "key is empty")
	util.AssertOk(task != nil, "task is nil")

	g.lock.Lock()

	if holder, ok := g.holders[key]; ok {
		g.lock.Unlock()
		holder.wg.Wait()
		return holder.result
	}

	holder := &resultHolder{}
	holder.wg.Add(1)
	g.holders[key] = holder
	g.lock.Unlock() //提前释放锁，避免阻塞其他任务

	holder.result = task.Run()
	holder.wg.Done()

	return holder.result
}

//获取key对应的缓存值 如果缓存值不存在返回nil
//如果正在执行对应的获取缓存值任务，则此方法将阻塞调用协程直到任务执行完成并返回缓存值
func (g *CacheGroup) Get(key string) Result {
	g.lock.RLock()
	defer g.lock.RUnlock()

	if holder, ok := g.holders[key]; ok {
		holder.wg.Wait()
		return holder.result
	}

	return nil
}

//删除key对应的缓存值
func (g *CacheGroup) Remove(key string) Result {
	g.lock.RLock()
	defer g.lock.RUnlock()

	if holder, ok := g.holders[key]; ok {
		delete(g.holders, key)
		return holder.result
	}

	return nil
}
