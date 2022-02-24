package rdb

import (
	"context"
	"github.com/bingooh/b-go-util/rdb"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func newRedisClient() *redis.Client {
	o := &redis.Options{
		Addr: "localhost:6379",
		DB:   3,
	}

	return redis.NewClient(o)
}

func TestRedisLimiter(t *testing.T) {
	r := require.New(t)

	client := newRedisClient()
	limiter := rdb.MustNewRateLimiter(`test_`, client)

	key := `test`
	ttl := 3 * time.Second
	ctx := context.Background()
	r.False(limiter.IsTooOften(ctx, key, ttl))
	r.True(limiter.IsTooOften(ctx, key, ttl))
	r.True(limiter.IsTooOften(ctx, key, ttl))

	time.Sleep(ttl) //等待key超时后删除
	r.False(limiter.IsTooOften(ctx, key, ttl))

	r.False(limiter.IsTooFrequent(ctx, key, 2, ttl))
	r.False(limiter.IsTooFrequent(ctx, key, 2, ttl))
	r.True(limiter.IsTooFrequent(ctx, key, 2, ttl)) //第3次调用，超过2次最大限制

	time.Sleep(ttl) //等待key超时后删除
	r.False(limiter.IsTooFrequent(ctx, key, 2, ttl))
}

func TestUtilCA(t *testing.T) {
	r := require.New(t)

	client := newRedisClient()
	ctx := context.Background()
	key := `test`

	assertEqual := func(expect interface{}) {
		v, err := client.Get(ctx, key).Int()
		r.NoError(err)
		r.EqualValues(expect, v)
	}

	assertExists := func(ok bool) {
		n, err := client.Exists(ctx, key).Result()
		r.NoError(err)

		expect := 0
		if ok {
			expect = 1
		}
		r.EqualValues(expect, n)
	}

	r.NoError(client.Del(ctx, key).Err())
	//r.NoError(rdb.Set(ctx,key,`false`,0).Err())

	//测试SetEQ，key不存在，则current==``
	ok, err := rdb.SetEQ(ctx, client, key, ``, 1)
	r.NoError(err)
	r.True(ok)
	assertEqual(1)

	ok, err = rdb.SetEQ(ctx, client, key, 2, 2)
	r.NoError(err)
	r.False(ok)
	assertEqual(1)

	ok, err = rdb.SetEQ(ctx, client, key, 1, 2)
	r.NoError(err)
	r.True(ok)
	assertEqual(2)

	//测试IncrEQ，current==2
	ok, val, err := rdb.IncrEQ(ctx, client, key, 1, 1)
	r.NoError(err)
	r.False(ok)
	r.EqualValues(2, val)
	assertEqual(2)

	ok, val, err = rdb.IncrEQ(ctx, client, key, 2, 1)
	r.NoError(err)
	r.True(ok)
	r.EqualValues(3, val)
	assertEqual(3)

	ok, val, err = rdb.IncrEQ(ctx, client, key, 3, -1)
	r.NoError(err)
	r.True(ok)
	r.EqualValues(2, val)
	assertEqual(2)

	//测试DelEQ，current==2
	ok, err = rdb.DelEQ(ctx, client, key, 1)
	r.NoError(err)
	r.False(ok)
	assertExists(true)

	ok, err = rdb.DelEQ(ctx, client, key, 2)
	r.NoError(err)
	r.True(ok)
	assertExists(false)

	//测试SetNEQ，key不存在，则current==``
	ok, err = rdb.SetNEQ(ctx, client, key, ``, 1)
	r.NoError(err)
	r.False(ok)
	assertExists(false)

	ok, err = rdb.SetNEQ(ctx, client, key, 0, 1)
	r.NoError(err)
	r.True(ok)
	assertEqual(1)

	ok, err = rdb.SetNEQ(ctx, client, key, 1, 2)
	r.NoError(err)
	r.False(ok)
	assertEqual(1)

	//测试IncrNEQ，current==1
	ok, val, err = rdb.IncrNEQ(ctx, client, key, 1, 1)
	r.NoError(err)
	r.False(ok)
	r.EqualValues(1, val)
	assertEqual(1)

	ok, val, err = rdb.IncrNEQ(ctx, client, key, 2, 1)
	r.NoError(err)
	r.True(ok)
	r.EqualValues(2, val)
	assertEqual(2)

	ok, val, err = rdb.IncrNEQ(ctx, client, key, 3, -1)
	r.NoError(err)
	r.True(ok)
	r.EqualValues(1, val)
	assertEqual(1)

}

func TestUtilCAH(t *testing.T) {
	r := require.New(t)

	client := newRedisClient()
	ctx := context.Background()
	key := `test`
	field := `f1`

	assertEqual := func(expect interface{}) {
		v, err := client.HGet(ctx, key, field).Int()
		r.NoError(err)
		r.EqualValues(expect, v)
	}

	assertExists := func(expect bool) {
		ok, err := client.HExists(ctx, key, field).Result()
		r.NoError(err)

		r.EqualValues(expect, ok)
	}

	r.NoError(client.Del(ctx, key).Err())

	//测试CAHSet，key不存在，则current==``
	ok, err := rdb.HSetEQ(ctx, client, key, field, ``, 1)
	r.NoError(err)
	r.True(ok)
	assertEqual(1)

	ok, err = rdb.HSetEQ(ctx, client, key, field, 2, 1)
	r.NoError(err)
	r.False(ok)
	assertEqual(1)

	ok, err = rdb.HSetEQ(ctx, client, key, field, 1, 2)
	r.NoError(err)
	r.True(ok)
	assertEqual(2)

	//测试CAHIncr,current==2
	ok, val, err := rdb.HIncrEQ(ctx, client, key, field, 1, 1)
	r.NoError(err)
	r.False(ok)
	r.EqualValues(0, val)
	assertEqual(2)

	ok, val, err = rdb.HIncrEQ(ctx, client, key, field, 2, 1)
	r.NoError(err)
	r.True(ok)
	r.EqualValues(3, val)
	assertEqual(3)

	ok, val, err = rdb.HIncrEQ(ctx, client, key, field, 3, -1)
	r.NoError(err)
	r.True(ok)
	r.EqualValues(2, val)
	assertEqual(2)

	//测试CAHIncr,current==2
	ok, err = rdb.HDelEQ(ctx, client, key, field, 1)
	r.NoError(err)
	r.False(ok)
	assertExists(true)

	ok, err = rdb.HDelEQ(ctx, client, key, field, 2)
	r.NoError(err)
	r.True(ok)
	assertExists(false)
}
