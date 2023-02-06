package rdb

import (
	"context"
	"github.com/bingooh/b-go-util/async"
	"github.com/go-redis/redis/v8"
	"github.com/lithammer/shortuuid/v4"
	"math/rand"
	"time"
)

var (
	//lua脚本变量缩写：v-缓存值，t-锁过期时间戳，o-锁持有者
	//lua脚本返回整数值数据类型为int64，如果返回0/1，建议转换为bool
	//如果脚本无返回值，则执行结果将返回redis.Nil

	//获取缓存，如果缓存值为空或者锁已过期，则加锁。返回`1`表示加锁成功，否则返回锁ttl(锁不存在则为nil)
	cacheGetScript = redis.NewScript(`
	local v = redis.call('HGET', KEYS[1], 'v')
	local t = redis.call('HGET', KEYS[1], 't')
	if t ~= false and tonumber(t) < tonumber(ARGV[2]) or t == false and v == false then
		redis.call('HSET', KEYS[1], 't', ARGV[3])
		redis.call('HSET', KEYS[1], 'o', ARGV[1])
		return  {v,'1'}
	end
	return {v, t}
    `)

	//更新缓存，如果参数owner不为空，则更新前校验是否仍然持有缓存锁
	cacheSetScript = redis.NewScript(`
	if ARGV[2] ~= '' then
		local o = redis.call('HGET', KEYS[1], 'o')
		if o ~= ARGV[2] then
			return 0
		end
	end
	redis.call('HSET', KEYS[1], 'v', ARGV[1])
	redis.call('HDEL', KEYS[1], 't')
	redis.call('HDEL', KEYS[1], 'o')
	redis.call('EXPIRE', KEYS[1], ARGV[3])
	return 1
    `)

	//失效缓存
	cacheExpireScript = redis.NewScript(`
	local exist = redis.call('EXISTS', KEYS[1])
	if exist == 1 then
		redis.call('HSET', KEYS[1], 't', 0)
		redis.call('HDEL', KEYS[1], 'o')
		redis.call('EXPIRE', KEYS[1], ARGV[1])
	end
	return 1
    `)

	//获取缓存锁，返回owner
	cacheLockScript = redis.NewScript(`
	local t = redis.call('HGET', KEYS[1], 't')
	local o = redis.call('HGET', KEYS[1], 'o')
	if t == false or tonumber(t) < tonumber(ARGV[2]) or o == ARGV[1] then
		redis.call('HSET', KEYS[1], 't', ARGV[3])
		redis.call('HSET', KEYS[1], 'o', ARGV[1])
		return ARGV[1]
	end
	return o
    `)

	//释放缓存锁，返回1表示成功
	cacheUnlockScript = redis.NewScript(`
	local o = redis.call('HGET', KEYS[1], 'o')
	if o == ARGV[1] then
		redis.call('HSET', KEYS[1], 't', 0)
		redis.call('HDEL', KEYS[1], 'o')
		return 1
	end
	return 0
    `)
)

type CacheItem struct {
	Key           string `json:"key"`
	Value         string `json:"value" redis:"v"`
	ExpiredAt     int64  `json:"expired_at"`
	LockOwner     string `json:"lock_owner" redis:"o"`
	LockExpiredAt int64  `json:"lock_expired_at" redis:"t"`
}

type CacheOption struct {
	LockTTL         time.Duration //锁TTL，应设置为最大缓存值计算时长，默认3s
	LockRetryDelay  time.Duration //重试获取锁的等待时长，默认100ms
	CacheTTLAdjust  float64       //缓存记录TTL调整因子，避免缓存同时失效，默认0.1
	EmptyCacheTTL   time.Duration //空缓存值TTL，设置为负数将不缓存空值，默认30秒
	ExpiredCacheTTL time.Duration //已失效缓存TTL，默认10s
}

func (o *CacheOption) MustNormalize() *CacheOption {
	if o.LockTTL <= 0 {
		o.LockTTL = 3 * time.Second
	}

	if o.LockRetryDelay <= 0 {
		o.LockRetryDelay = 100 * time.Millisecond
	}

	if o.CacheTTLAdjust <= 0 {
		o.CacheTTLAdjust = 0.1
	}

	if o.EmptyCacheTTL == 0 {
		o.EmptyCacheTTL = 30 * time.Second
	}

	if o.ExpiredCacheTTL <= 0 {
		o.ExpiredCacheTTL = 10 * time.Second
	}

	return o
}

type Cache struct {
	option *CacheOption
	client redis.UniversalClient
	group  *async.CacheGroup
	onLoad func(ctx context.Context, key string) (string, error) //查询最新缓存值回调函数
}

func NewCache(option *CacheOption, client redis.UniversalClient, onLoad func(ctx context.Context, key string) (string, error)) *Cache {
	return &Cache{
		option: option.MustNormalize(),
		client: client, onLoad: onLoad,
		group: async.NewCacheGroup(),
	}
}

// Fetch 获取缓存值(最终一致性)。如果缓存值已失效，则返回旧值并异步查询最新值
func (c *Cache) Fetch(ctx context.Context, key string, cacheTTL time.Duration) (string, error) {
	return c.fetchInCacheGroup(key, func() (string, error) {
		owner := shortuuid.New()
		val, lock, err := c.runScriptGet(ctx, key, owner)

		//如果值为空且获取锁失败，则不断重试获取锁
		for err == nil && val == nil && lock != `1` {
			time.Sleep(c.option.LockRetryDelay)
			val, lock, err = c.runScriptGet(ctx, key, owner)
		}

		if err != nil {
			return ``, err
		}

		if lock != `1` {
			//锁为空(返回最新值)或被他人获取(返回旧值)
			return val.(string), nil
		}

		//说明已获取锁，如果值为空则同步查询返回最新值，否则异步查询返回当前旧值
		if val == nil {
			return c.load(ctx, key, owner, cacheTTL)
		}

		go c.load(ctx, key, owner, cacheTTL)

		return val.(string), nil
	})
}

// FetchNew 获取缓存值(强一致性)。如果缓存值已失效，则同步查询并返回最新值
func (c *Cache) FetchNew(ctx context.Context, key string, cacheTTL time.Duration) (string, error) {
	return c.fetchInCacheGroup(key, func() (string, error) {
		owner := shortuuid.New()
		val, lock, err := c.runScriptGet(ctx, key, owner)

		//如果锁不为空且未获取到锁，则不断重试获取锁
		for err == nil && lock != nil && lock != `1` {
			time.Sleep(c.option.LockRetryDelay)
			val, lock, err = c.runScriptGet(ctx, key, owner)
		}

		if err != nil {
			return ``, err
		}

		if lock != `1` {
			//锁为空(返回最新值)，不可能被他人获取，否则前面代码会不断重试获取锁
			return val.(string), nil
		}

		//说明已获取锁，同步查询返回最新值
		return c.load(ctx, key, owner, cacheTTL)
	})
}

// FetchBackend 直接查询后端获取缓存值，不更新缓存
func (c *Cache) FetchBackend(ctx context.Context, key string) (string, error) {
	return c.fetchInCacheGroup(key, func() (string, error) {
		return c.onLoad(ctx, key)
	})
}

func (c *Cache) fetchInCacheGroup(key string, fn func() (string, error)) (string, error) {
	//fn执行期间进入的后续查询请求将被阻塞等待第1个请求查询结束。fn执行完成后删除group里key对应缓存，以便后续查询将能访问到redis/backend
	return c.group.Do(key, async.ToValTask(func() (val interface{}, err error) {
		val, err = fn()
		//考虑是否sleep一段时间，以便更多后续请求可以获取cache group里的缓存
		c.group.Del(key)
		return
	})).String()
}

func (c *Cache) load(ctx context.Context, key, owner string, cacheTTL time.Duration) (string, error) {
	val, err := c.onLoad(ctx, key)
	if err != nil {
		_, _ = c.runScriptUnlock(ctx, key, owner)
		return ``, err
	}

	if val == `` {
		if c.option.EmptyCacheTTL < 0 {
			err = c.Del(ctx, key)
			return ``, err
		}

		cacheTTL = c.option.EmptyCacheTTL
	}

	err = c.runScriptSet(ctx, key, val, owner, cacheTTL)
	return val, err
}

// Expire 失效缓存(标记删除)
func (c *Cache) Expire(ctx context.Context, key string) error {
	return c.runScriptExpire(ctx, key)
}

// Del 删除缓存
func (c *Cache) Del(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// Set 设置缓存,不检查缓存锁
func (c *Cache) Set(ctx context.Context, key, val string, cacheTTL time.Duration) error {
	return c.runScriptSet(ctx, key, val, ``, cacheTTL)
}

// Get 获取缓存值
func (c *Cache) Get(ctx context.Context, key string) (val string, exist bool, err error) {
	val, err = c.client.HGet(ctx, key, `v`).Result()
	if err == nil {
		exist = true
	} else if err == redis.Nil {
		err = nil
	}

	return
}

// GetCacheItem 获取缓存记录
func (c *Cache) GetCacheItem(ctx context.Context, key string) (item *CacheItem, err error) {
	rs := c.client.HGetAll(ctx, key)
	if len(rs.Val()) == 0 {
		return nil, nil
	}

	item = &CacheItem{Key: key}
	err = c.client.HGetAll(ctx, key).Scan(item)

	if err == nil {
		if v := c.client.TTL(ctx, key).Val(); v > 0 {
			item.ExpiredAt = time.Now().Add(v).Unix()
		}
	}

	return
}

func (c *Cache) runScriptGet(ctx context.Context, key, owner string) (val, lock interface{}, err error) {
	var result interface{}
	result, err = cacheGetScript.Run(ctx, c.client, []string{key}, owner, time.Now().Unix(), time.Now().Add(c.option.LockTTL).Unix()).Result()
	if err == nil {
		rs := result.([]interface{})
		val = rs[0]
		lock = rs[1]
	}

	return
}

func (c *Cache) runScriptSet(ctx context.Context, key, val, owner string, cacheTTL time.Duration) error {
	cacheTTL -= time.Duration(rand.Float64() * c.option.CacheTTLAdjust * float64(cacheTTL))
	ttl := int(cacheTTL / time.Second)
	return cacheSetScript.Run(ctx, c.client, []string{key}, val, owner, ttl).Err()
}

func (c *Cache) runScriptLock(ctx context.Context, key, owner string, lockTTL time.Duration) (bool, error) {
	rs, err := cacheLockScript.Run(ctx, c.client, []string{key}, owner, time.Now().Unix(), time.Now().Add(lockTTL).Unix()).Result()
	if err != nil {
		return false, err
	}

	return rs == owner, nil
}

func (c *Cache) runScriptUnlock(ctx context.Context, key, owner string) (bool, error) {
	return cacheUnlockScript.Run(ctx, c.client, []string{key}, owner).Bool()
}

func (c *Cache) runScriptExpire(ctx context.Context, key string) error {
	return cacheExpireScript.Run(ctx, c.client, []string{key}, int(c.option.ExpiredCacheTTL/time.Second)).Err()
}
