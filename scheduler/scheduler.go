package scheduler

import (
	"context"
	"github.com/bingooh/b-go-util/conf"
	"github.com/bingooh/b-go-util/util"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"sync"
	"time"
)

type Scheduler struct {
	option *Option
	logger *zap.Logger
	lock   sync.Mutex

	cr         *cron.Cron
	cronLogger cron.Logger
	tasks      map[string]cron.EntryID //key为任务名称
}

//读取默认配置文件scheduler.toml创建Scheduler
func MustNewSchedulerFromDefaultCfg() *Scheduler {
	option := &Option{}
	conf.MustScanConfFile(option, `scheduler.toml`)
	return MustNewScheduler(option)
}

func MustNewScheduler(option *Option) *Scheduler {
	s := &Scheduler{
		option:     option.MustNormalize(),
		logger:     newSchedulerZapLogger(),
		cronLogger: newSchedulerLogger(),
		tasks:      make(map[string]cron.EntryID),
	}

	opts := []cron.Option{cron.WithLogger(s.cronLogger)}
	if option.EnableCronSeconds {
		opts = append(opts, cron.WithSeconds())
	}

	s.cr = cron.New(opts...)
	return s
}

func (s *Scheduler) HasTask(name string) bool {
	s.lock.Lock()
	defer s.lock.Unlock()

	_, ok := s.tasks[name]
	return ok
}

func (s *Scheduler) HasTaskOption(name string) bool {
	return s.option.Task(name) != nil
}

func (s *Scheduler) MustAddTaskFn(name string, task func()) {
	s.MustAddTask(name, cron.FuncJob(task))
}

func (s *Scheduler) MustAddTask(name string, task cron.Job) {
	util.AssertNotEmpty(name, `name为空`)
	util.AssertOk(task != nil, `task为空`)

	opt := s.option.Task(name)
	util.AssertOk(opt != nil, `任务配置[%v]不存在`, name)

	s.lock.Lock()
	defer s.lock.Unlock()

	_, exist := s.tasks[name]
	util.AssertOk(!exist, `任务[%v]已存在`, name)

	//每个job单独定义调用链，未使用全局定义
	//var job cron.Job = cron.FuncJob(task)
	if opt.EnableRecover {
		task = cron.Recover(s.cronLogger)(task)
	}

	if opt.SkipIfStillRunning {
		task = cron.SkipIfStillRunning(s.cronLogger)(task)
	} else if opt.DelayIfStillRunning {
		task = cron.DelayIfStillRunning(s.cronLogger)(task)
	}

	id, err := s.cr.AddJob(opt.Cron, task)
	util.AssertNilErr(err, `新增任务[%v]出错`, name)

	s.tasks[name] = id
	s.logger.Sugar().Infof(`新增任务[%v]`, name)
}

func (s *Scheduler) RemoveTasks(names ...string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, name := range names {
		if id, ok := s.tasks[name]; ok {
			s.cr.Remove(id)
			delete(s.tasks, name)
			s.logger.Sugar().Infof(`删除任务[%v]`, name)
		}
	}
}

func (s *Scheduler) Start() {
	s.cr.Start()
	s.logger.Info(`scheduler已启动`)
}

//此方法会阻塞当前线程，直到超时或所有任务停止执行
func (s *Scheduler) Stop(timeout time.Duration) {
	if timeout <= 0 {
		<-s.cr.Stop().Done()
		s.logger.Info(`scheduler已停止`)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case <-s.cr.Stop().Done():
		s.logger.Info(`scheduler已停止`)
	case <-ctx.Done():
		s.logger.Warn(`scheduler停止超时`)
	}
}
