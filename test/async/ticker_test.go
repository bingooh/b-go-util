package async

import (
	"context"
	"fmt"
	"github.com/bingooh/b-go-util/async"
	"github.com/bingooh/b-go-util/util"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestTicker(t *testing.T) {
	//r:=require.New(t)

	printlnTick := func(tip string, ticker *async.Ticker) {
		fmt.Println(tip, `------------------`)
		for tm := range ticker.C {
			fmt.Println(tm.Format(`04:05`))
		}
		fmt.Println(time.Now().Format(`04:05`), `done`)
	}

	incrCount := func(ticker *async.Ticker, n, count int, sleep time.Duration) {
		for i := 0; i < n; i++ {
			time.Sleep(sleep)
			ticker.IncrCount(count)
		}
		ticker.Close()
	}

	//每1秒触发1次tick，未设置minCount/maxCount，建议直接使用time.Ticker
	ticker1 := async.NewTicker(async.NewTickerOption(0, 0, 1*time.Second))
	go incrCount(ticker1, 7, 1, 500*time.Millisecond)
	printlnTick(`ticker1`, ticker1)

	//每3秒触发1次tick,count>=maxCount才触发
	ticker2 := async.NewTicker(async.NewTickerOption(0, 3, 0))
	go incrCount(ticker2, 7, 1, 1*time.Second)
	printlnTick(`ticker2`, ticker2)

	//每1秒触发1次tick,count>=maxCount才触发
	ticker3 := async.NewTicker(async.NewTickerOption(0, 1, 0))
	go incrCount(ticker3, 3, 1, 1*time.Second)
	printlnTick(`ticker3`, ticker3)

	//每3秒触发1次tick,count>=minCount才触发
	ticker4 := async.NewTicker(async.NewTickerOption(3, 3, 1*time.Second))
	go incrCount(ticker4, 7, 1, 1*time.Second)
	printlnTick(`ticker4`, ticker4)

	//每1秒触发1次tick,count>=maxCount才触发
	ticker5 := async.NewTicker(async.NewTickerOption(1, 1, 3*time.Second))
	go incrCount(ticker5, 7, 1, 1*time.Second)
	printlnTick(`ticker5`, ticker5)

	//每1秒触发1次tick,count>=maxCount触发。虽然period设置为每3秒触发一次，但是距上次触发时间也需间隔3秒才会触发
	ticker6 := async.NewTicker(async.NewTickerOption(0, 1, 3*time.Second))
	go incrCount(ticker6, 7, 1, 1*time.Second)
	printlnTick(`ticker6`, ticker6)
}

func TestTickerExecutor(t *testing.T) {
	r := require.New(t)

	handle := func(tasks []interface{}) {
		fmt.Printf("%v[size=%v,tasks=%v]\n", time.Now().Format(`0405`), len(tasks), tasks)
	}

	fmt.Println(`ex1 -------------`)
	ex1 := async.NewTickerExecutor(async.NewTickerOption(1, 1, 0), handle)
	ex1.Add(1, 1)
	ex1.Add(1, 2)
	time.Sleep(1 * time.Millisecond) //等待任务执行
	r.EqualValues(0, ex1.TaskSize())
	ex1.Close()

	fmt.Println(`ex2 -------------`)
	ex2 := async.NewTickerExecutor(async.NewTickerOption(0, 3, 0), handle)
	ex2.Add(1, 1)
	ex2.Close() //关闭会立刻执行已添加任务
	r.EqualValues(0, ex2.TaskSize())
	ex2.Add(1, 2) //已关闭，添加任务直接返回
	r.EqualValues(0, ex2.TaskSize())

	fmt.Println(`ex3 -------------`)
	ex3 := async.NewTickerExecutor(async.NewTickerOption(0, 3, 1*time.Second), handle)
	for i := 1; i <= 2; i++ {
		ex3.Add(i, i)
		ex3.InvokeNow() //立刻执行，不会等待tick
	}
	r.EqualValues(0, ex3.TaskSize())
	ex3.Add(1, 3)
	time.Sleep(2 * time.Second)
	r.EqualValues(0, ex3.TaskSize())
	ex3.Close()

	fmt.Println(`ex4 -------------`)
	total := util.NewAtomicInt64(0)
	ex4 := async.NewTickerExecutor(async.NewTickerOption(10, 100, 3*time.Second), func(tasks []interface{}) {
		total.Incr(int64(len(tasks)))
		handle(tasks)
	})

	g := async.NewWorkerGroup(10)
	for i := 0; i < 10; i++ {
		g.Run(func() {
			for j := 0; j < 100; j++ {
				ex4.Add(1, j)

				if j < 90 {
					time.Sleep(util.RandDuration(100, 200) * time.Millisecond)
				} else {
					time.Sleep(util.RandDuration(1, 3) * time.Second)
				}
			}
		})
	}
	g.Wait()
	ex4.Close() //会立刻处理剩余数据
	r.EqualValues(0, ex4.TaskSize())
	r.EqualValues(1000, total.Value())
}

func TestTickerCount1(t *testing.T) {
	//r:=require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	source := make(chan int)
	defer close(source)

	async.RunCancelable(ctx, func() {
		for i := 1; i <= 120; i++ {
			if ctx.Err() != nil {
				return
			}

			source <- i
			if i <= 100 {
				time.Sleep(100 * time.Millisecond)
			} else {
				time.Sleep(1 * time.Second)
			}
		}
	})

	ticker := async.NewTicker(async.NewTickerOption(1, 10, 3*time.Second))
	defer ticker.Close()

	count := 0
	values := make([]int, 0)
	for {
		select {
		case <-ctx.Done():
			//source可能包含未处理的数据，应考虑使用drain()
			fmt.Println(`done`)
			return
		case v := <-source:
			count++
			values = append(values, v)
			ticker.IncrCount(1)
		case tm := <-ticker.C:
			fmt.Printf("%v[count=%v,size=%v]\n", tm.Format(`0405`), count, len(values))
			count = 0
			values = values[0:0]
		}
	}
}

func TestTickerCount2(t *testing.T) {
	ticker := async.NewTicker(async.NewTickerOption(10, 100, 3*time.Second))
	defer ticker.Close()

	var lock sync.Mutex

	values := make([]int, 0)

	add := func() {
		lock.Lock()
		defer lock.Unlock()

		values = append(values, 1)
		ticker.IncrCount(1)
	}

	handle := func(tm time.Time) {
		lock.Lock()
		defer lock.Unlock()

		fmt.Printf("%v[size=%v]\n", tm.Format(`0405`), len(values))

		ticker.ResetCount()
		values = values[0:0]
	}

	go func() {
		for tm := range ticker.C {
			handle(tm)
		}
	}()

	g := async.NewWorkerGroup(10)
	for i := 0; i < 10; i++ {
		g.Run(func() {
			for j := 0; j < 100; j++ {
				add()

				if j < 80 {
					time.Sleep(util.RandDuration(100, 200) * time.Millisecond)
				} else {
					time.Sleep(util.RandDuration(1, 3) * time.Second)
				}
			}
		})
	}
	g.Wait()

	fmt.Println(`done`)
	handle(time.Now()) //处理剩余数据
}
