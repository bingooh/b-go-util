package rdb

import (
	"fmt"
	"github.com/bingooh/b-go-util/rdb/scheduler"
	"github.com/bingooh/b-go-util/util"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestScheduler(t *testing.T) {
	r := require.New(t)

	newTask := func(name string, sleep time.Duration) func(ctx scheduler.Context) error {
		return func(ctx scheduler.Context) error {
			if sleep > 0 {
				time.Sleep(sleep)
			}
			util.Log(`do task: %v`, name)
			return nil
		}
	}

	s1 := scheduler.MustNewSchedulerFromDefaultCfgFile()
	r.Panics(func() {
		//任务名称必须与已有任务选项匹配，否则崩溃
		s1.MustAddTaskFn(`txx`, newTask(`txx`, 0))
	})

	s1.MustAddTaskFn(`t1`, newTask(`t1`, 5*time.Second)) //t1定时每秒执行1次，但任务本身耗时5秒，实际每5秒执行1次
	s1.MustAddTaskFn(`t2`, newTask(`t2`, 0))             //t2已禁用，不会添加到调度器
	s1.Start()

	//启动另1个定时器，执行任务t1
	//2个任务t1会竞争分布式锁，所以仍然是大概每5秒执行1次
	s2 := scheduler.MustNewSchedulerFromDefaultCfgFile()
	s2.MustAddTaskFn(`t1`, newTask(`t1`, 5*time.Second))
	s2.Start()

	time.Sleep(20 * time.Second)
	s1.Stop()
	s2.Stop()
	fmt.Println(`done1`)

	//t3指定下次执行时间，实际每3秒执行1次
	s3 := scheduler.MustNewSchedulerFromDefaultCfgFile()
	s3.MustAddTaskFn(`t3`, func(ctx scheduler.Context) error {
		return ctx.SetTaskNextInvokeTime(time.Now().Add(3 * time.Second))
	})
	s3.Start()
	time.Sleep(10 * time.Second)
	s3.Stop()
	fmt.Println(`done2`)

}
