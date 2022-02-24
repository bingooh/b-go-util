package rdb

import (
	"b-go-util/rdb"
	"context"
	"fmt"
	"github.com/bsm/redislock"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestRedisSessionLock(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	key := `test.lock.1`
	client := newRedisClient()
	r.NoError(client.Del(ctx, key).Err())

	locker1 := rdb.NewLocker(newRedisClient())
	locker2 := rdb.NewLocker(newRedisClient())

	//locker1获取会话锁，锁的TTL设置为1秒。但自动续约，所以不会超时
	lock, err := locker1.ObtainSessionLock(ctx, key, 1*time.Second)
	r.NoError(err)
	r.False(lock.IsReleased())

	//等待2秒，locker2仍然获取不到锁
	time.Sleep(2 * time.Second)
	_, err = locker2.ObtainSessionLock(ctx, key, 1*time.Second)
	r.True(err == redislock.ErrNotObtained)

	//释放锁后，locker2可以成功获取
	r.NoError(lock.Release(ctx))
	r.True(lock.IsReleased())
	lock, err = locker2.ObtainSessionLock(ctx, key, 1*time.Second)
	r.NoError(err)
	r.NoError(lock.Release(ctx))
	r.True(lock.IsReleased())

	//测试重试获取锁
	isLockRefreshFail := false //锁是否续约失败
	r.NoError(client.Del(ctx, key).Err())
	r.NoError(client.Set(ctx, key, `1`, 5*time.Second).Err()) //先占有锁，5秒后释放锁。lock3将在5秒后获取锁

	time.Sleep(1 * time.Second)
	locker3 := rdb.NewLocker(newRedisClient()).
		WithRetryOption(10, 1*time.Second).                                         //每秒1次重试10次
		WithOnLockRefreshFailed(func(lockName string) { isLockRefreshFail = true }) //锁续约失败回调函数

	start := time.Now()
	lock, err = locker3.ObtainSessionLock(ctx, key, 1*time.Second)
	r.NoError(err)
	r.False(lock.IsReleased())
	r.WithinDuration(time.Now(), start.Add(5*time.Second), 1*time.Second)

	//占有锁，以便让lock续约失败
	r.NoError(client.Set(ctx, key, `1`, 5*time.Second).Err())
	time.Sleep(2 * time.Second) //等待时间必须大于lockTTL
	r.True(isLockRefreshFail)   //续约失败会调用回调函数
	r.True(lock.IsReleased())

	//启动多个协程竞争锁，获取锁的协程将持有1秒然后释放
	//如果下1协程获取锁的间隔小于1秒，则测试失败
	r.NoError(client.Del(ctx, key).Err())
	n := 10
	var wg sync.WaitGroup
	wg.Add(n)

	var lastLockObtainTime time.Time
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()

			locker := rdb.NewLocker(newRedisClient())
			for {
				//锁的ttl为100ms
				lock, err := locker.ObtainSessionLock(ctx, key, 100*time.Millisecond)
				if err == redislock.ErrNotObtained {
					time.Sleep(1 * time.Second)
					continue
				}

				fmt.Println(`get lock:`, i)
				r.True(lastLockObtainTime.IsZero() || time.Since(lastLockObtainTime) >= 1*time.Second)

				lastLockObtainTime = time.Now()
				time.Sleep(1 * time.Second)

				fmt.Println(`release lock:`, i)
				r.NoError(lock.Release(ctx))
				r.True(lock.IsReleased())

				return
			}

		}()
	}
	wg.Wait()

}
