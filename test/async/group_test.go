package async

import (
	"errors"
	"fmt"
	"github.com/bingooh/b-go-util/async"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

var assertFailErr = errors.New(`assert fail`)

func TestWaitGroup(t *testing.T) {
	r := require.New(t)

	g := async.NewWaitGroup()
	start := time.Now()

	g.Wait() //未添加任务不会阻塞
	r.True(time.Since(start) < 1*time.Millisecond)

	//添加3个任务，任务最长沉睡3秒结束。总共耗时3秒
	for i := 1; i <= 3; i++ {
		g.Run(newJob(i))
	}
	g.Wait()

	r.WithinDuration(time.Now(), start.Add(3*time.Second), 1*time.Second)
}

func TestRoutineGroup(t *testing.T) {
	r := require.New(t)

	start := time.Now()
	g := async.NewRoutineGroup(3, newJob(3)) //启动3个协程执行同1个任务，任务沉睡3秒后结束
	g.Wait()                                 //未启动任务不会阻塞
	r.True(time.Since(start) < 1*time.Millisecond)

	g.Start() //不会阻塞线程
	g.Wait()  //任务总共耗时3秒
	r.WithinDuration(time.Now(), start.Add(3*time.Second), 1*time.Second)

	//g等待上次执行完成后可以再次执行，但不建议这样使用
	fmt.Println(`--------------`)
	g.Run() //再耗时3秒
	r.WithinDuration(time.Now(), start.Add(6*time.Second), 1*time.Second)
}

func TestWorkerGroup(t *testing.T) {
	r := require.New(t)

	start := time.Now()
	g := async.NewWorkerGroup(1)
	g.Wait() //未添加任务不会阻塞
	r.True(time.Since(start) < 1*time.Millisecond)

	//添加3个任务，任务最长沉睡3秒结束。任务并发数为1，总共耗时6秒
	fmt.Println(`--------------`)
	for i := 1; i <= 3; i++ {
		g.Run(newJob(i))
	}
	g.Wait()
	r.WithinDuration(time.Now(), start.Add(6*time.Second), 1*time.Second)

	//添加3个任务，任务最长沉睡3秒结束。任务并发数为3，总共耗时3秒
	start = time.Now()
	g = async.NewWorkerGroup(3)
	fmt.Println(`--------------`)
	for i := 1; i <= 3; i++ {
		g.Run(newJob(i))
	}
	g.Wait()
	r.WithinDuration(time.Now(), start.Add(3*time.Second), 1*time.Second)

	//g可以重用，以下总共耗时7秒
	start = time.Now()
	fmt.Println(`--------------`)
	for i := 1; i <= 5; i++ {
		g.Run(newJob(i))
	}
	g.Wait()
	fmt.Println(time.Since(start))
	r.WithinDuration(time.Now(), start.Add(7*time.Second), 1*time.Second)
}
