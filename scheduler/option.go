package scheduler

import "github.com/bingooh/b-go-util/util"

type Option struct {
	EnableCronSeconds bool                   //是否启用秒定时设置
	Tasks             map[string]*TaskOption //key为任务名称
}

//任务选项。每个任务单独配置，不支持全局配置
type TaskOption struct {
	Cron                string //任务触发时间，参考：https://github.com/robfig/cron
	EnableRecover       bool   //任务执行崩溃后是否恢复，默认为true
	SkipIfStillRunning  bool   //任务正在执行是否跳过本次触发。默认为true。如果设置为true，则DelayIfStillRunning设置无效
	DelayIfStillRunning bool   //任务正在执行是否延迟本次触发
}

func (o *Option) MustNormalize() *Option {
	util.AssertOk(o != nil, `option为空`)

	for name, task := range o.Tasks {
		util.AssertOk(task != nil, `task为空`)
		util.AssertNotEmpty(task.Cron, `cron为空[task=%v]`, name)

		task.EnableRecover = true
		task.SkipIfStillRunning = true
	}

	return o
}

func (o *Option) Task(name string) *TaskOption {
	if len(o.Tasks) > 0 {
		return o.Tasks[name]
	}

	return nil
}
