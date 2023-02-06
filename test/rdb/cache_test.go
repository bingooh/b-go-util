package rdb

import (
	"context"
	"github.com/bingooh/b-go-util/rdb"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	key := `test.cache.1`
	cacheVal := `1`
	cacheLoadCount := 0

	client := newRedisClient()
	r.NoError(client.Del(ctx, key).Err())

	option := &rdb.CacheOption{}
	c1 := rdb.NewCache(option, client, func(ctx context.Context, key string) (string, error) {
		cacheLoadCount++
		return cacheVal, nil
	})

	v1, exist, err := c1.Get(ctx, key)
	r.NoError(err)
	r.True(v1 == `` && !exist)

	item1, err := c1.GetCacheItem(ctx, key)
	r.NoError(err)
	r.Nil(item1)

	//cache ttl<3s，会减去1个随机值，避免多个缓存同时失效
	r.NoError(c1.Set(ctx, key, cacheVal, 3*time.Second))
	v1, exist, err = c1.Get(ctx, key)
	r.NoError(err)
	r.True(v1 == cacheVal && exist)

	item1, err = c1.GetCacheItem(ctx, key)
	r.NoError(err)
	r.EqualValues(key, item1.Key)
	r.EqualValues(cacheVal, item1.Value)
	r.True(item1.ExpiredAt > 0)
	r.Empty(item1.LockOwner)
	r.EqualValues(0, item1.LockExpiredAt)

	r.NoError(c1.Del(ctx, key))
	v1, exist, err = c1.Get(ctx, key)
	r.NoError(err)
	r.True(v1 == `` && !exist)

	//测试fetch
	v2, err := c1.Fetch(ctx, key, 3*time.Second)
	r.NoError(err)
	r.EqualValues(cacheVal, v2)
	r.EqualValues(1, cacheLoadCount)

	item2, err := c1.GetCacheItem(ctx, key)
	r.NoError(err)
	r.EqualValues(key, item2.Key)
	r.EqualValues(cacheVal, item2.Value)
	r.True(item2.ExpiredAt > 0)
	r.Empty(item2.LockOwner)
	r.EqualValues(0, item2.LockExpiredAt)

	time.Sleep(1 * time.Second)
	v2, err = c1.Fetch(ctx, key, 3*time.Second)
	r.NoError(err)
	r.EqualValues(cacheVal, v2)      //缓存未失效
	r.EqualValues(1, cacheLoadCount) //未重新加载缓存

	time.Sleep(2 * time.Second)
	v2, exist, err = c1.Get(ctx, key)
	r.NoError(err)
	r.True(v2 == `` && !exist) //缓存已失效

	//测试expire
	r.NoError(c1.Expire(ctx, key))
	item3, err := c1.GetCacheItem(ctx, key)
	r.NoError(err)
	r.Nil(item3) //key不存在，调用c1.Expire()无影响

	r.NoError(c1.Set(ctx, key, cacheVal, 30*time.Second))
	r.NoError(c1.Expire(ctx, key))
	item3, err = c1.GetCacheItem(ctx, key)
	r.NoError(err)
	r.EqualValues(cacheVal, item3.Value) //缓存值仍然存在
	r.EqualValues(0, item3.LockExpiredAt)
	r.Empty(item3.LockOwner)
	r.True(item3.ExpiredAt <= time.Now().Add(option.ExpiredCacheTTL).Unix()) //设置缓存失效将改变缓存记录ttl，默认10s

	//缓存失效后，再次获取会查询新值
	oldCacheVal := cacheVal
	cacheVal = `2`
	c3, err := c1.Fetch(ctx, key, 3*time.Second)
	r.NoError(err)
	r.EqualValues(oldCacheVal, c3) //fetch异步查询，返回旧值

	time.Sleep(500 * time.Millisecond)
	c3, _, err = c1.Get(ctx, key)
	r.NoError(err)
	r.EqualValues(cacheVal, c3)
	r.EqualValues(2, cacheLoadCount)

	//再次失效缓存，测试fetchNew()
	r.NoError(c1.Expire(ctx, key))
	cacheVal = `3`
	c3, err = c1.FetchNew(ctx, key, 3*time.Second)
	r.NoError(err)
	r.EqualValues(cacheVal, c3) //fetchNew同步查询，返回新值
	r.EqualValues(3, cacheLoadCount)

	//fetchBackend()直接查询后端，不更新缓存
	oldCacheVal = cacheVal
	cacheVal = `4`
	c3, err = c1.FetchBackend(ctx, key)
	r.NoError(err)
	r.EqualValues(cacheVal, c3)
	c3, _, err = c1.Get(ctx, key)
	r.NoError(err)
	r.EqualValues(oldCacheVal, c3) //缓存值未更新
}
