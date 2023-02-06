package scheduler

import (
	"github.com/bingooh/b-go-util/async"
	"github.com/bingooh/b-go-util/conf"
	"github.com/bingooh/b-go-util/rdb"
	"github.com/bingooh/b-go-util/slog"
	"github.com/bingooh/b-go-util/util"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"strconv"
	"time"
)

type Task interface {
	Run(ctx Context) error
}

type TaskFn func(ctx Context) error

func (f TaskFn) Run(ctx Context) error {
	return f(ctx)
}

var DefaultScheduler *Scheduler

func MustGetDefaultScheduler() *Scheduler {
	util.AssertOk(DefaultScheduler != nil, `defaultScheduler为空`)
	return DefaultScheduler
}

func MustInitDefaultSchedulerFromDefaultCfgFile() *Scheduler {
	if DefaultScheduler == nil {
		DefaultScheduler = MustNewSchedulerFromDefaultCfgFile()
	}

	return DefaultScheduler
}

// MustNewSchedulerFromDefaultCfgFile 读取默认配置文件scheduler.toml创建Scheduler
func MustNewSchedulerFromDefaultCfgFile() *Scheduler {
	option := &Option{}
	conf.MustLoad(option, `scheduler`)
	return MustNewScheduler(option)
}

func MustNewScheduler(option *Option) *Scheduler {
	return &Scheduler{
		option:    option.MustNormalize(),
		logger:    slog.NewLogger(`scheduler`),
		isRunning: util.NewAtomicBool(false),
		tasks:     make(map[string]Task),
	}
}

// Scheduler 任务调度器
// 每个任务将定时调用，每次调用需要通过以下全部检测才会真正执行任务
// - 当前时间是否处于任务执行时间区间
// - 当前时间是否大于任务下次执行时间
// - 启用任务锁且成功获取任务锁(redis分布式锁)
type Scheduler struct {
	option *Option
	logger *zap.Logger

	client    *redis.Client
	isRunning *util.AtomicBool
	tasks     map[string]Task //key为任务名称
}

func (s *Scheduler) MustAddTaskFn(name string, task TaskFn) {
	s.MustAddTask(name, task)
}

// MustAddTask 添加任务，参数name必须与任务配置项名称相匹配
func (s *Scheduler) MustAddTask(name string, task Task) {
	util.AssertOk(s.isRunning.False(), `scheduler已启动`)
	util.AssertNotEmpty(name, `name为空`)
	util.AssertOk(task != nil, `task为空`)

	if s.option.MustGetTaskOption(name).Disabled {
		return
	}

	_, exist := s.tasks[name]
	util.AssertOk(!exist, `task已存在[name=%v]`, name)

	s.tasks[name] = task
	s.logger.Sugar().Infof(`添加任务[%v]`, name)
}

func (s *Scheduler) Start() {
	if s == nil || !s.isRunning.CASwap(false) {
		return
	}

	if len(s.tasks) == 0 {
		s.logger.Info(`未启用任何任务`)
		return
	}

	s.client = redis.NewClient(s.option.Redis)

	for name, task := range s.tasks {
		s.runTask(s.option.MustGetTaskOption(name), task)
	}
}

func (s *Scheduler) Stop() {
	if s == nil || !s.isRunning.CASwap(true) {
		return
	}

	s.option.rootCtx.Cancel()
	time.Sleep(3 * time.Second)

	if s.client != nil {
		if err := s.client.Close(); err != nil {
			s.logger.Error(`redis关闭出错`, zap.Error(err))
		}
	}
}

func (s *Scheduler) runTask(o *TaskOption, task Task) {
	ctx := s.option.RootContext()
	logger := slog.NewLogger(`task`, o.name)
	taskCtx := &BaseContext{
		option: o, ctx: ctx,
		client: s.client, logger: logger,
	}

	var locker *rdb.Locker
	if !o.DisableTaskLock {
		locker = rdb.NewLocker(s.client)
	}

	isInInvokeTimeRange := func() bool {
		if len(o.InvokeTimeRange) == 0 {
			return true
		}

		now, err := strconv.Atoi(time.Now().Format(`1504`))
		util.AssertNilErr(err, `当前日期转换为HHMI出错`)

		return o.InvokeTimeRange[0] <= now && now <= o.InvokeTimeRange[1]
	}

	//下次执行任务时间是否已到
	isTaskNextInvokeTimeUp := func() (bool, error) {
		v, err := taskCtx.GetTaskNextInvokeTime()
		if err != nil {
			return false, err
		}

		return v.IsZero() || time.Since(v) >= 0, nil
	}

	obtainLock := func() (*rdb.SessionLock, error) {
		if locker != nil {
			return locker.ObtainSessionLock(ctx, o.TaskLockKey(), o.TaskLockTTL)
		}

		return nil, nil
	}

	releaseLock := func(lock *rdb.SessionLock) {
		if lock != nil {
			if err := lock.Release(ctx); err != nil {
				logger.Error(`会话锁释放失败`, zap.String(`key`, o.TaskLockKey()), zap.Error(err))
			}
		}
	}

	async.RunCancelableInterval(ctx, o.InvokeInternal, func(c async.Context) {
		if c.Done() || s.isRunning.False() {
			return
		}

		if !isInInvokeTimeRange() {
			return
		}

		if ok, err := isTaskNextInvokeTimeUp(); err != nil || !ok {
			if err != nil {
				logger.Error(`任务下次执行时间查询出错，等待下次重试`, zap.String(`key`, o.TaskInvokeTimeKey()), zap.Error(err))
			}
			return
		}

		lock, err := obtainLock()
		if err != nil {
			logger.Error(`会话锁获取失败，等待下次重试`, zap.String(`key`, o.TaskLockKey()), zap.Error(err))
			return
		}
		defer releaseLock(lock)

		if err = task.Run(taskCtx); err != nil {
			logger.Error(`任务执行出错`, zap.Error(err))
			return
		}

		next, _ := taskCtx.GetTaskNextInvokeTime()
		logger.Info(`任务执行完成`, zap.Time(`下次执行时间`, next))
	})
}
