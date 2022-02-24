package scheduler

import (
	"b-go-util/scheduler"
	"b-go-util/util"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestScheduler(t *testing.T) {
	r := require.New(t)

	newTask := func(name string, sleep time.Duration) func() {
		return func() {
			if sleep > 0 {
				time.Sleep(sleep)
			}
			util.Log(`do task: %v`, name)
		}
	}

	s := scheduler.MustNewSchedulerFromDefaultCfg()
	r.Panics(func() {
		//任务名称必须与已有任务选项匹配，否则崩溃
		s.MustAddTaskFn(`txx`, newTask(`txx`, 0))
	})

	s.MustAddTaskFn(`t1`, newTask(`t1`, 0))
	s.MustAddTaskFn(`t2`, newTask(`t2`, 0))
	s.MustAddTaskFn(`t5`, newTask(`t5`, 0))
	s.Start()

	//添加已存在的任务将崩溃
	r.True(s.HasTask(`t1`))
	r.True(s.HasTaskOption(`t1`))
	r.Panics(func() {
		//添加已存在的同名任务将崩溃
		s.MustAddTaskFn(`t1`, newTask(`t1`, 0))
	})

	time.Sleep(10 * time.Second)
	s.RemoveTasks(`t1`, `t2`, `t3`, `t5`)

	//任务t1每秒执行1次，但每次执行耗时2秒，所以会被跳过。等同于每3秒执行1次
	s.MustAddTaskFn(`t1`, newTask(`t1`, 2*time.Second))
	time.Sleep(10 * time.Second)

	//添加1个崩溃任务，默认会回复执行
	s.MustAddTaskFn(`t2`, func() {
		panic(`t2 panic`)
	})
	time.Sleep(5 * time.Second)

	s.Stop(0)
	time.Sleep(10 * time.Second)
}
