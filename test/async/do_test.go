package async

import (
	"b-go-util/async"
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestDoTimeLimitTask(t *testing.T) {
	//DoXXX()使用当前协程执行任务函数

	//以下任务耗时1秒，超时时间2秒，任务正常完成
	start := time.Now()
	ctx := async.DoTimeLimitTask(2*time.Second, func() {
		time.Sleep(1 * time.Second)
		fmt.Println("task1 done")
	})
	require.False(t, ctx.Timeout())                                                //ctx未超时
	require.WithinDuration(t, time.Now(), start.Add(1*time.Second), 1*time.Second) //耗时1秒

	//等待2秒后，ctx仍未超时，即使内部使用的context.Context已超时取消
	time.Sleep(2 * time.Second)
	require.False(t, ctx.Timeout())
}

func TestDoUntilCancel(t *testing.T) {
	//以下任务总共耗时3秒，任务函数总共被回调4次，ctx取消后回调最后1次
	start := time.Now()
	ctx, cancel := context.WithCancel(context.Background())

	//循环执行任务直到ctx取消
	async.DoUntilCancel(ctx, func(c async.Context) {
		if c.Done() {
			fmt.Println("task done:", c.Count())
			return
		}

		fmt.Println("task do:", c.Count())
		time.Sleep(1 * time.Second)

		//任务执行3次后，取消外部ctx->c.Done()为true
		if c.Count() == 3 {
			cancel()

			//也可调用此方法主动取消任务（终止循环)
			//主动取消不会回调最后1次，具体见TestRunUtilCancel()
			//c.Abort()
		}
	})

	require.WithinDuration(t, time.Now(), start.Add(3*time.Second), 1*time.Second) //耗时3秒
	require.True(t, ctx.Err() == context.Canceled)
}
