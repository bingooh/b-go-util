package async

import (
	"b-go-util/async"
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestRunCancelableInterval(t *testing.T) {
	//async.RunCancelableInterval()内部使用time.Ticker定时执行任务

	//以下任务每3秒执行1次，1秒后超时取消
	//因内部使用ticker，外部ctx在1秒后超时可立刻取消任务
	//注1:如果任务内部使用了time.Sleep()，则仍然无法立刻取消（见以下测试）
	//注2：async.RunUtilCancel()将会在下次循环才能检查到外部ctx超时取消
	start := time.Now()
	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
	<-async.RunCancelableInterval(ctx, 3*time.Second, func(c async.Context) {
		if c.Done() {
			fmt.Println("task1 done", time.Now().Second())
			return
		}

		//以下代码不会执行，因此首次执行时外部ctx超时，上面代码已返回
		fmt.Println("task1 start", time.Now().Second())
	})
	require.WithinDuration(t, time.Now(), start.Add(1*time.Second), 1*time.Second)

	//以下任务定时每1秒执行1次，重试3次后主动取消执行任务
	//以下总共耗时:1秒+3次*2秒=6秒，首次等待1秒，每次执行任务耗时2秒，导致ticker下次触发延时
	count := int64(0)
	start = time.Now()
	<-async.RunCancelableInterval(context.Background(), 1*time.Second, func(c async.Context) {
		count = c.Count()

		if c.Done() {
			//主动取消，不会执行以下代码
			fmt.Println("task2 done")
			return
		}

		//c.Count()从1开始计数
		time.Sleep(2 * time.Second)
		fmt.Println("task2 do: ", c.Count())

		if c.Count() == 3 {
			c.Abort()
		}
	})
	require.EqualValues(t, 3, count)
	require.WithinDuration(t, time.Now(), start.Add(7*time.Second), 1*time.Second)

	//以下任务耗时3秒，每秒执行1次，3秒后超时
	//async.RunTimeLimitInterval()简单封装async.RunCancelableInterval()
	start = time.Now()
	<-async.RunTimeLimitInterval(1*time.Second, 3*time.Second, func(ctx async.Context) {
		if ctx.Done() {
			fmt.Println("task3 done: ", ctx.Count())
			//可能出现以下特殊情况
			//- 第3次执行->超时->执行以下代码，此时ctx.Count==4
			//- 第2次执行->超时->执行以下代码，此时ctx.Count==3
			//综上：不要依赖tick时间和超时时间计算执行次数，如果要实现retry逻辑，应使用ctx.Abort()
			require.True(t, ctx.Count() == 3 || ctx.Count() == 4)
			require.True(t, ctx.Timeout())
			require.False(t, ctx.Canceled())
			require.False(t, ctx.Aborted())
			return
		}

		fmt.Println("task3 do: ", ctx.Count())
	})
	require.WithinDuration(t, time.Now(), start.Add(3*time.Second), 1*time.Second)
}
