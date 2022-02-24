package async

import (
	"b-go-util/async"
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

//创建定时任务，简单打印执行次数
func newIntervalTask(prefix string) func(ctx async.Context) {
	return func(ctx async.Context) {
		if ctx.Done() {
			fmt.Println(prefix, "done:", ctx.Count())
			return
		}

		fmt.Println(prefix, "do:", ctx.Count())
	}
}

//创建定时任务，简单打印执行次数。同时将执行次数设置给count参数
func newIntervalCountTask(prefix string, count *int64) func(ctx async.Context) {
	return func(ctx async.Context) {
		*count = ctx.Count()

		if ctx.Done() {
			fmt.Println(prefix, "done:", ctx.Count())
			return
		}

		fmt.Println(prefix, "do:", ctx.Count())
	}
}

func TestRunIntervalHelper(t *testing.T) {
	//任务耗时2秒,外部ctx超时
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	start := time.Now()
	<-async.NewRunIntervalHelper(1 * time.Second).
		WithContext(ctx). //设置外部ctx
		Run(newIntervalTask("task1"))
	require.WithinDuration(t, time.Now(), start.Add(2*time.Second), 1*time.Second)

	//任务耗时1秒，自定义1秒超时
	ctx, _ = context.WithTimeout(context.Background(), 3*time.Second)
	start = time.Now()
	<-async.NewRunIntervalHelper(1 * time.Second).
		WithContext(ctx).
		WithTimeout(1 * time.Second). //设置超时时间，比外部ctx先超时
		Run(newIntervalTask("task2"))
	require.WithinDuration(t, time.Now(), start.Add(1*time.Second), 1*time.Second)

	//任务执行1次，首次执行延时2秒，3秒后超时
	start = time.Now()
	count := int64(0)
	<-async.NewRunIntervalHelper(2 * time.Second).
		WithTimeout(3 * time.Second).      //设置3秒超时
		WithInitRunDelay(2 * time.Second). //设置首次执行延时2秒
		Run(newIntervalCountTask("task3", &count))
	require.WithinDuration(t, time.Now(), start.Add(3*time.Second), 1*time.Second)
	require.EqualValues(t, 2, count)

	//任务执行4次(重试3次)，retry内部使用ctx.Abort()结束定时循环，因此退出时不会再次回调任务函数
	count = int64(0)
	<-async.NewRunIntervalHelper(1 * time.Second).
		WithMaxRetryCount(3). //设置重试3次
		Run(newIntervalCountTask("task4", &count))
	require.EqualValues(t, 4, count) //定时循环结束后不会再回调任务函数，否则count应为5

	//任务函数传入的async.Context在执行期间不会done，以避免if ctx.Done(){}被执行2次
	<-async.NewRunIntervalHelper(1 * time.Second).
		WithTimeout(3 * time.Second). //设置3秒超时，内部使用context.Context实现定时
		Run(func(ctx async.Context) {
			//如果ctx在执行过程中可以done，则以下代码可能执行2次
			if ctx.Done() {
				fmt.Println("task5 done")
				return
			}

			//3秒后超时，这里等待4秒，ctx.Done()仍然返回false
			fmt.Println("task5 do")
			require.False(t, ctx.Done())
			time.Sleep(4 * time.Second)
			require.False(t, ctx.Done())
		})
}
