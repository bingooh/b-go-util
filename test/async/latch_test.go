package async

import (
	"fmt"
	"github.com/bingooh/b-go-util/async"
	"github.com/stretchr/testify/require"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func startTasks(latch *async.Latch, n int, task func(i int, latch *async.Latch)) {
	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			task(i, latch)
		}()
	}

	wg.Wait()
}

func TestLatch(t *testing.T) {
	r := require.New(t)

	latch := async.NewLatch(false)
	r.False(latch.IsClosed())
	r.False(latch.CAOpen())
	r.False(latch.IsClosed())

	latch.Close()
	r.True(latch.IsClosed())
	r.False(latch.CAClose())
	r.True(latch.IsClosed())

	latch.Open()
	r.False(latch.IsClosed())

	//latch当前未关闭，期望已关闭，状态不同所以切换失败返回false
	r.False(latch.CASwap(true))
	r.False(latch.IsClosed())

	//latch当前未关闭，期望未关闭，状态相同所以切换成功返回true
	r.True(latch.CASwap(false))
	r.True(latch.IsClosed())

	r.True(latch.CAOpen())
	r.False(latch.IsClosed())

	r.True(latch.CAClose())
	r.True(latch.IsClosed())

	start := time.Now()
	latch.Open()
	latch.Wait() //不应阻塞
	r.WithinDuration(time.Now(), start, 10*time.Millisecond)

	//阻塞3秒
	start = time.Now()
	latch.Close()
	time.AfterFunc(3*time.Second, latch.Open)
	startTasks(latch, 10, func(i int, latch *async.Latch) {
		latch.Wait()
	})
	r.WithinDuration(time.Now(), start.Add(3*time.Second), 1*time.Second)

	//每隔一定时长切换latch状态，观察是否触发崩溃错误

	//启动多个协程随机等待一定时长切换latch状态，观察是否触发崩溃错误
	go startTasks(latch, 10, func(i int, latch *async.Latch) {
		for {
			time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)

			if rand.Intn(10)%3 == 0 {
				latch.Open()
			} else {
				latch.Close()
			}
		}
	})

	//每个协程等待latch打开等待随机时长后执行任务
	startTasks(latch, 1000, func(i int, latch *async.Latch) {
		c := 0
		for {
			c++

			if c%100 == 0 {
				fmt.Printf("do(%v):%v\n", i, c)
			}

			if c >= 1000 {
				fmt.Printf("done(%v)\n", i)
				return
			}

			latch.Wait()
			time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
		}
	})
}
