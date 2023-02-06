package async

import (
	"fmt"
	"github.com/bingooh/b-go-util/async"
	"github.com/bingooh/b-go-util/util"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestTimerExecutor(t *testing.T) {
	r := require.New(t)
	start := util.NewAtomicTime()

	printTask := func(key, value interface{}) {
		fmt.Printf("task[key=%v,elapsed=%v,val=%v]\n", key, time.Since(start.Value()), value)
	}

	ex1 := async.NewTimerExecutor(2*time.Second, 5, func(key, value interface{}) {
		printTask(key, value)
		delay := time.Duration(key.(int)) * time.Second
		r.WithinDuration(time.Now(), start.Value().Add(delay), 2100*time.Millisecond) //误差为period+误差值
	})

	//ex1创建后内部ticker即开始计时，如果新增任务的delay<ex1.period，则任务将会添加到当前bucket，tick事件触发后任务将被立刻执行
	//如以下任务1（key==1），其执行延迟时长为0~2s。任务1可能加入ex1后就触发tick事件从而立刻执行
	//即每个任务实际延迟执行时长为(即误差范围0~period)：
	// - 如果task.delay< executor.period: 0~executor.period
	// - 如果task.delay>=executor.period: task.delay~task.delay+executor.period
	//time.Sleep(1*time.Second)//延迟添加任务，以实现添加任务因触发tick事件立刻执行
	fmt.Println(`ex1---------------`)
	start.Set(time.Now())
	ex1.Put(1*time.Second, 1)
	ex1.Put(2*time.Second, 2)
	ex1.Put(4*time.Second, 4)
	ex1.Put(10*time.Second, 10)
	time.Sleep(13 * time.Second)

	start.Set(time.Now())
	ex1.Put(3*time.Second, 3)
	ex1.Close() //将丢弃未执行的任务，不会阻塞
	r.False(ex1.Put(1*time.Second, 100))
	r.WithinDuration(time.Now(), start.Value(), 10*time.Millisecond)
	ex1.Close() //重复关闭不会报错
	time.Sleep(6 * time.Second)

	fmt.Println(`ex2---------------`)
	ex2 := async.NewTimerExecutor(1*time.Second, 10, func(key, value interface{}) {
		printTask(key, value)
		r.EqualValues(1, key)
		r.WithinDuration(time.Now(), start.Value().Add(3*time.Second), 2100*time.Millisecond) //新任务可能延迟(3+1+1)s
	})
	start.Set(time.Now())
	ex2.Put(2*time.Second, 1)
	time.Sleep(1 * time.Second)
	ex2.Put(3*time.Second, 1) //旧任务将被删除，新任务将重新计算延迟时长
	ex2.Put(2*time.Second, 2)
	ex2.Del(2) //task2不会执行
	time.Sleep(5 * time.Second)

	ex2.WithOnClosedHandler(func(key, value interface{}) {
		fmt.Println(`pending task`)
		printTask(key, value)
		time.Sleep(2 * time.Second)
		fmt.Println(`ex2 closed`)
	})
	r.True(ex2.Put(10*time.Second, 10))
	ex2.Close() //将会等待OnClosed()执行完成后关闭

	fmt.Println(`ex3---------------`)
	ex3 := async.NewTimerExecutor(1*time.Second, 5, printTask)
	start.Set(time.Now())
	ex3.Put(1*time.Second, 1)
	ex3.PutTask(2*time.Second, 2, 2, true) //循环执行任务
	time.Sleep(10 * time.Second)
	ex3.Close()
	time.Sleep(5 * time.Second)
}

func addTimeoutTask(fn func(delay time.Duration)) {
	n := 10
	size := 100

	g := async.NewWaitGroup()
	for i := 0; i < n; i++ {
		g.Run(func() {
			for j := 0; j < size; j++ {
				fn(util.RandDuration(1, 3) * time.Second)
			}
		})
	}
	g.Wait()
}

func BenchmarkTimerExecutor(b *testing.B) {
	ex := async.NewTimerExecutor(1*time.Second, 1000, func(key, value interface{}) {})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		addTimeoutTask(func(delay time.Duration) {
			ex.Put(delay, i)
		})
	}
}

func BenchmarkTimerExecutorStd(b *testing.B) {
	fn := func() {}
	for i := 0; i < b.N; i++ {
		addTimeoutTask(func(delay time.Duration) {
			time.AfterFunc(delay, fn)
		})
	}
}
