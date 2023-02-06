package async

import (
	"fmt"
	"github.com/bingooh/b-go-util/async"
	"github.com/stretchr/testify/require"
	"log"
	"testing"
	"time"
)

func TestBarrier(t *testing.T) {
	r := require.New(t)

	newJob := func(i int, sleep time.Duration) func() {
		return func() {
			log.Println(`task`, i)
			time.Sleep(sleep)
		}
	}

	//获取锁后执行1次函数，每个任务耗时1秒，总共耗时5秒
	start := time.Now()
	b1 := async.NewBarrier()
	g := async.NewWaitGroup()
	for i := 0; i < 5; i++ {
		i := i
		g.Run(func() {
			b1.Do(newJob(i, 1*time.Second)) //等待获取锁后执行回调函数
		})
	}
	g.Wait()
	r.WithinDuration(time.Now(), start.Add(5*time.Second), 1*time.Second)

	fmt.Println(`--------------------`)

	//1秒内最多执行1次函数，每个任务耗时0秒，总共耗时5秒
	start = time.Now()
	b2 := async.NewPeriodBarrier(1 * time.Second)
	for i := 0; i < 50; i++ {
		i := i
		g.Run(func() {
			b2.Do(newJob(i, 0)) //如果不满足执行条件，直接返回
		})

		time.Sleep(100 * time.Millisecond)
	}
	g.Wait()
	r.WithinDuration(time.Now(), start.Add(5*time.Second), 1*time.Second)
}
