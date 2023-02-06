package async

import (
	"github.com/bingooh/b-go-util/async"
	"github.com/bingooh/b-go-util/util"
	"github.com/stretchr/testify/require"
	"strconv"
	"sync"
	"testing"
	"time"
)

// 创建获取缓存值任务，任务沉睡1秒后返回val作为缓存值，同时count++
func newFetchCacheTask(val string, count *util.AtomicInt64) async.Task {
	return async.ToValTask(func() (interface{}, error) {
		time.Sleep(1 * time.Second)
		if count != nil {
			count.Incr(1)
		}

		return val, nil
	})
}

func TestCacheGroupDo(t *testing.T) {
	count := util.NewAtomicInt64(0)

	//添加100个不同key的缓存，获取缓存的任务耗时1秒。创建n个线程获取缓存值
	//期望结果:仅有100个协程执行获取缓存任务，总共耗时1秒。其他协程直接获取缓存值
	g := async.NewCacheGroup()

	var taskDoneWg sync.WaitGroup
	var startWg sync.WaitGroup

	n := 10000
	taskDoneWg.Add(n)
	startWg.Add(1)

	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer taskDoneWg.Done()
			startWg.Wait() //等待全部协程初始化完成

			//创建缓存的key及其对应的获取缓存值的任务
			//对于相同的key，仅在对应缓存值不存在的情况下，CacheGroup才会执行
			//第1个请求添加的任务去获取缓存值，其他请求直接返回第1个请求获取的缓存值
			key := strconv.FormatInt(int64(i%100), 10)
			result := g.Do(key, newFetchCacheTask(key, count))
			require.Equal(t, key, result.MustString())
		}()
	}
	startWg.Done()

	start := time.Now()
	taskDoneWg.Wait()
	require.EqualValues(t, 100, count.Value())
	require.WithinDuration(t, time.Now(), start.Add(1*time.Second), 1*time.Second)
}

func TestCacheGroupDel(t *testing.T) {
	key := "1"
	g := async.NewCacheGroup()

	result := g.Do(key, newFetchCacheTask(key, nil))
	require.Equal(t, key, result.MustString())

	result = g.Get(key) //获取现有缓存值，如无返回nil
	require.Equal(t, key, result.MustString())

	result = g.Del(key) //删除缓存值，返回被删除的值
	require.Equal(t, key, result.MustString())

	result = g.Get(key)
	require.Nil(t, result) //删除后再次获取缓存结果为nil

	//创建1个获取缓存值任务，需要等待3秒后返回缓存值
	task := async.ToValTask(func() (interface{}, error) {
		time.Sleep(3 * time.Second)
		return key, nil
	})

	var c1Wg sync.WaitGroup
	var c2Wg sync.WaitGroup
	var c3Wg sync.WaitGroup
	c1Wg.Add(1)
	c2Wg.Add(1)
	c3Wg.Add(1)

	c1 := async.Run(func() {
		c1Wg.Done()
		result := g.Do(key, task)
		require.Equal(t, key, result.MustString())
	})

	c2 := async.Run(func() {
		c1Wg.Wait()
		time.Sleep(1 * time.Second) //等待c1调用group.Do()后获取缓存值
		c2Wg.Done()

		result := g.Get(key)
		require.Equal(t, key, result.MustString()) //c2仍然可获取缓存值
	})

	c3 := async.Run(func() {
		c2Wg.Wait()
		time.Sleep(1 * time.Second) //等待c2调用group.Get()后删除缓存值
		c3Wg.Done()

		result := g.Del(key)
		require.Nil(t, result) //不能获取缓存值，c1还未执行完任务
	})

	c4 := async.Run(func() {
		c3Wg.Wait()
		time.Sleep(3 * time.Second) //等待c3调用group.Del()后且c1执行完任务后再获取缓存值

		result := g.Get(key)
		require.Nil(t, result) //不能获取缓存值，c3已将缓存删除
	})

	_, _, _, _ = <-c1, <-c2, <-c3, <-c4
}

func TestCacheGroupDelMulti(t *testing.T) {
	r := require.New(t)

	key := `1`
	count := util.NewAtomicInt64(0)
	g := async.NewCacheGroup()

	doGet := func() {
		wg := async.NewWaitGroup()
		for i := 0; i < 10; i++ {
			wg.Run(func() {
				//回调函数执行期间后续进入的请求都将等待获取结果
				val := g.Do(key, async.ToValTask(func() (interface{}, error) {
					time.Sleep(1 * time.Second)
					count.Incr(1)
					r.Nil(g.Del(key)) //移除缓存
					return key, nil
				})).MustString()

				r.Equal(key, val)
			})
		}
		wg.Wait()
	}

	doGet()
	r.EqualValues(1, count.Value())
	r.Nil(g.Get(key))

	doGet()
	r.EqualValues(2, count.Value())
	r.Nil(g.Get(key))
}
