package async

import (
	"fmt"
	"github.com/bingooh/b-go-util/async"
	"github.com/bingooh/b-go-util/util"
	"github.com/stretchr/testify/require"
	"math/rand"
	"sync"
	"testing"
	"time"
)

//向pool添加n个任务，每个任务沉睡sleep时长后结束
func addTasks(pool async.Pool, n int, sleep time.Duration) {
	for i := 0; i < n; i++ {
		pool.Run(func() {
			time.Sleep(sleep)
		})
	}
}

//向pool添加n个任务，每个任务随机沉睡(0-100)*sleep时长后结束
func addRandSleepTasks(pool async.Pool, n int, sleep time.Duration) {
	for i := 0; i < n; i++ {
		pool.Run(func() {
			time.Sleep(time.Duration(rand.Intn(100)) * sleep)
		})
	}
}

//捕获协程池崩溃异常，成功捕获设置hasCaught为true
func catchPoolWaitPanic(pool async.Pool, hasCaught *util.AtomicBool) {
	defer util.OnExit(func(err error) {
		if err != nil {
			hasCaught.CASwap(false)
		}
	})

	pool.Wait()
}

func TestPool(t *testing.T) {
	//协程池，用于限制执行任务的协程数，进而限制系统资源消耗
	//以下声明最多无阻塞添加6个任务：4 worker+2 task in queue
	//最多同时执行4个任务，其中2个协程一直运行，另2个协程空闲10秒后退出
	pool := async.NewWorkerPool(2, 4, 2)

	//添加6个任务,不会阻塞。每个任务沉睡3秒后结束
	start := time.Now()
	addTasks(pool, 6, 3*time.Second)
	require.WithinDuration(t, time.Now(), start, 1*time.Second)
	require.EqualValues(t, 6, pool.PendingTaskSize())

	//添加第7个任务将会阻塞3秒，等待其中1个任务执行完成
	start = time.Now()
	addTasks(pool, 1, 0)
	require.WithinDuration(t, time.Now(), start.Add(3*time.Second), 1*time.Second)

	pool.Wait()                                       //等待任务执行完成
	require.EqualValues(t, 0, pool.PendingTaskSize()) //所有任务执行完成
	require.EqualValues(t, 4, pool.ExistWorkerSize()) //当前共4个协程

	//等待11秒，让2个worker因空闲超时(10秒超时)退出
	time.Sleep(11 * time.Second)
	require.EqualValues(t, 2, pool.ExistWorkerSize()) //当前共2个协程

	//pool.Wait()可多次等待,以下共添加6个任务，每个任务沉睡2秒后结束，总共耗时4秒
	start = time.Now()
	addTasks(pool, 3, 2*time.Second)
	pool.Wait()
	addTasks(pool, 3, 2*time.Second)
	pool.Wait()
	require.WithinDuration(t, time.Now(), start.Add(4*time.Second), 1*time.Second)
}

func TestPoolClose(t *testing.T) {
	pool := async.NewWorkerPool(2, 4, 2)

	//关闭池将丢弃任务队列里待处理任务
	//如果任务是无限循环或需要等待很长时间，则Close()将一直等待
	addTasks(pool, 4, 2*time.Second) //添加4个2秒完成任务，此任务将由worker执行
	addTasks(pool, 2, 6*time.Second) //添加2个6秒完成任务，此任务将在队列里等待

	start := time.Now()
	pool.Close()                                                                   //立刻丢弃2个在队列里的任务，一直等待其余4个任务完成（将在2秒内完成）
	require.WithinDuration(t, time.Now(), start.Add(2*time.Second), 1*time.Second) //关闭耗时2秒
	require.EqualValues(t, 0, pool.PendingTaskSize())                              //全部任务完成
	require.EqualValues(t, 0, pool.ExistWorkerSize())                              //全部协程退出

	//pool.CloseWithTimeout()与pool.Close()类似，但在关闭超时后退出阻塞当前协程，同时pool.Wait()也将退出阻塞
	//注意：关闭超时不会停止正在执行任务的后台协程，这可能导致协程泄漏
	//如果关闭超时，可调用pool.PendingTaskCount()获取待处理的任务数

	//以下添加6个耗时15秒的任务，但关闭池在3秒后超时
	pool = async.NewWorkerPool(2, 4, 2)
	addTasks(pool, 6, 15*time.Second) //添加6个任务，2个任务在队列里

	start = time.Now()
	pool.CloseWithTimeout(3 * time.Second)                                         //立刻丢弃2个在队列里的任务，3秒后关闭超时，退出阻塞
	pool.Wait()                                                                    //不再阻塞
	require.WithinDuration(t, time.Now(), start.Add(3*time.Second), 1*time.Second) //关闭耗时3秒
	require.EqualValues(t, 4, pool.PendingTaskSize())                              //剩下4个待处理的任务阻塞池的协程，可能造成协程泄漏

	time.Sleep(11 * time.Second)
	require.EqualValues(t, 4, pool.ExistWorkerSize()) //等待11秒后，任务仍未完成，4个协程仍未退出

	time.Sleep(5 * time.Second)
	require.EqualValues(t, 0, pool.PendingTaskSize()) //全部任务完成
	require.EqualValues(t, 0, pool.ExistWorkerSize()) //全部协程退出
}

func TestPoolWait(t *testing.T) {
	//pool.Wait()会一直阻塞直到当前池里的全部任务执行完成，或者池已关闭
	//pool.Wait()仅支持当前协程在添加任务后调用，不支持当前协程添加任务，另1协程等待，否则可能报错，具体解释见poo.Wait()注释说明

	//以下循环200次，每次添加10000个任务，并等待任务执行完成
	pool := async.NewWorkerPool(20, 1000, 10)
	for i := 0; i < 200; i++ {
		addRandSleepTasks(pool, 10000, 1*time.Nanosecond)
		pool.Wait() //如果报错会崩溃
	}
	pool.Close()

	//以下2个不同协程调用pool.Wait()，可能在任何1处调用时报错
	pool = async.NewWorkerPool(20, 1000, 10)
	hasWaitErr := util.NewAtomicBool(false)
	for {
		//不断循环直到捕捉到pool.Wait()崩溃错误
		if hasWaitErr.True() {
			break
		}
		addRandSleepTasks(pool, 10000, 1*time.Nanosecond)

		go func() {
			catchPoolWaitPanic(pool, hasWaitErr) //调用pool.Wait()并捕捉崩溃错误
		}()

		catchPoolWaitPanic(pool, hasWaitErr) //调用pool.Wait()并捕捉崩溃错误
	}
	pool.Close()
	require.True(t, hasWaitErr.Value())

	//如果其他协程的确需要监听任务完成，可使用以下方式
	pool = async.NewWorkerPool(20, 1000, 10)
	for i := 0; i < 200; i++ {
		//使用wg通知其他协程全部任务完成
		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			wg.Wait()
		}()

		addRandSleepTasks(pool, 10000, 1*time.Nanosecond)
		pool.Wait()
		wg.Done()
	}
	pool.Close()
}

func TestPoolRunRecursive(t *testing.T) {
	pool := async.NewWorkerPool(2, 4, 0)

	//在协程池执行任务时，向协程池新增1个任务，将可能导致死锁
	for i := 0; i < 200; i++ {
		i := i
		pool.Run(func() {
			<-pool.Run(func() {
				fmt.Println("task done:", i)
			})
		})
	}

	pool.Wait()
	pool.Close()
}
