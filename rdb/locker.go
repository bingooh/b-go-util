package rdb

import (
	"b-go-util/async"
	"b-go-util/util"
	"context"
	"github.com/bsm/redislock"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"time"
)

//会话锁续约失败回调函数
type OnLockRefreshFailed func(lockName string)

//创建锁
type Locker struct {
	*redislock.Client

	retryLimit            int
	retryInterval         time.Duration
	onLockRefreshFailedFn OnLockRefreshFailed
}

func NewLocker(client *redis.Client) *Locker {
	return &Locker{
		Client: redislock.New(client),
	}
}

//设置锁续约失败回调函数
func (l *Locker) WithOnLockRefreshFailed(fn OnLockRefreshFailed) *Locker {
	l.onLockRefreshFailedFn = fn
	return l
}

//设备获取锁
func (l *Locker) WithRetryOption(limit int, interval time.Duration) *Locker {
	l.retryLimit = limit
	l.retryInterval = interval
	return l
}

//获取锁
func (l *Locker) obtainLock(ctx context.Context, lockName string, lockTTL time.Duration) (lock *redislock.Lock, err error) {
	//由于redislock库的问题(redislock.go/64行)，lockTTL参数值将设置给ctx作为其超时时间
	//假设设置lockTTL=1秒，重试策略为每秒重试1次，最多10次。则实际会在1秒后返回获取锁失败，即lockTTL超时导致获取锁失败
	//以下自定义重试逻辑
	if l.retryLimit <= 0 {
		return l.Client.Obtain(ctx, lockName, lockTTL, nil)
	}

	if l.retryInterval <= 0 {
		l.retryInterval = lockTTL
	}

	helper := async.NewRunIntervalHelper(l.retryInterval).
		WithMaxRetryCount(int64(l.retryLimit)).WithInitRunDelay(1 * time.Millisecond).WithContext(ctx)

	<-helper.Run(func(c async.Context) {
		if c.Done() {
			return
		}

		lock, err = l.Client.Obtain(ctx, lockName, lockTTL, nil)
		if err == nil || err != redislock.ErrNotObtained {
			c.Abort()
		}
	})

	if err == nil && lock == nil {
		err = redislock.ErrNotObtained
	}

	return
}

//获取会话锁，会话锁会自动续约锁
func (l *Locker) ObtainSessionLock(ctx context.Context, lockName string, lockTTL time.Duration) (*SessionLock, error) {
	lock, err := l.obtainLock(ctx, lockName, lockTTL)
	if err != nil {
		return nil, err
	}

	return newSessionLock(lock, lockTTL, l.onLockRefreshFailedFn), nil
}

//自动续约锁(会话锁)
type SessionLock struct {
	*redislock.Lock

	logger              *zap.Logger
	lockTTL             time.Duration
	isReleased          *util.AtomicBool
	refreshRunner       *async.Runner
	onLockRefreshFailed OnLockRefreshFailed
}

func newSessionLock(lock *redislock.Lock, lockTTL time.Duration, onLockRefreshFailed OnLockRefreshFailed) *SessionLock {
	s := &SessionLock{
		logger:              newLogger(`session_lock`),
		Lock:                lock,
		lockTTL:             lockTTL,
		isReleased:          util.NewAtomicBool(false),
		onLockRefreshFailed: onLockRefreshFailed,
	}

	s.refreshRunner = async.NewRunner(async.BgTaskFn(s.refreshBgTask)).MustStart()
	return s
}

func (s *SessionLock) Release(ctx context.Context) error {
	if !s.isReleased.CASwap(false) {
		return nil
	}

	s.refreshRunner.Stop()
	return s.Lock.Release(ctx)
}

func (s *SessionLock) ReleaseWithTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return s.Release(ctx)
}

func (s *SessionLock) IsReleased() bool {
	return s == nil || s.isReleased.Value()
}

func (s *SessionLock) refreshBgTask(ctx context.Context) (<-chan struct{}, error) {
	//在锁到期前2秒续约，如果锁的TTL小于2秒，则每隔ttl/2时长进行续约
	refreshInterval := s.lockTTL - 2*time.Second
	if refreshInterval <= 0 {
		refreshInterval = s.lockTTL / 2
	}

	return async.RunCancelableInterval(ctx, refreshInterval, func(c async.Context) {
		if c.Done() {
			return
		}

		err := s.Lock.Refresh(ctx, s.lockTTL, nil)
		if err == nil || c.Done() || s.isReleased.True() {
			return
		}

		s.isReleased.Set(true)
		s.logger.Error(`锁续约失败`, zap.String(`lock`, s.Lock.Key()), zap.String(`token`, s.Lock.Token()), zap.Error(err))
		c.Abort()

		if s.onLockRefreshFailed != nil {
			s.onLockRefreshFailed(s.Lock.Key())
		}

	}), nil
}
