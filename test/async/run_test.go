package async

import (
	"context"
	"fmt"
	"github.com/bingooh/b-go-util/async"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
	"time"
)

func TestRunPattern(t *testing.T) {
	//串行/并行设计模式

	//创建3个任务，分别沉睡1-3秒后执行结束
	job1, job2, job3 := newJob(1), newJob(2), newJob(3)

	//串行执行，耗时6秒
	start := time.Now()
	<-async.Run(job1)
	<-async.Run(job2)
	<-async.Run(job3)
	require.WithinDuration(t, time.Now(), start.Add(6*time.Second), 1*time.Second, `串行耗时6秒`)

	//并行执行,耗时3秒
	start = time.Now()
	c1, c2, c3 := async.Run(job1), async.Run(job2), async.Run(job3)

	_, _, _ = <-c1, <-c2, <-c3 //注意：必须写在同1行
	require.WithinDuration(t, time.Now(), start.Add(3*time.Second), 1*time.Second, `并行耗时3秒`)
}

func TestRunTask(t *testing.T) {
	//RunXX(task)     执行无需返回值的任务
	//RunXXTask(task) 执行需要返回值的任务

	//以下是1个不好的用法，使用1个外部变量获取任务返回值，且未使用lock/atomic
	n := 0
	<-async.Run(func() {
		n = 1
	})
	require.Equal(t, 1, n, "n should equal 1")

	//以下是正确用法，使用RunTask()获取任务返回值
	result := <-async.RunTask(async.ToValTask(func() (interface{}, error) {
		return 2, nil
	}))
	require.Equal(t, 2, result.MustInt(), "n should equal 2")

	//result提供多个util方法对返回值做简单数据类型转换
	//以下将报错，因为无法将整型转换为字符串，需自行转换
	_, err := result.String()
	require.Error(t, err, "should has type cast err")

	//自行将结果转换为字符串
	str1 := fmt.Sprintf("%v", result.Value())
	str2 := strconv.Itoa(result.MustInt())
	require.Equal(t, "2", str1)
	require.Equal(t, "2", str2)
}

func TestRunTimeLimitTask(t *testing.T) {
	//以下任务耗时1秒，超时时间3秒，任务正常完成
	result := <-async.RunTimeLimitTask(3*time.Second, newTask(1))
	require.NoError(t, result.Error())
	require.False(t, result.Timeout())
	require.Equal(t, 1, result.MustInt())

	//任务耗时5秒，超时时间1秒，任务超时
	result = <-async.RunTimeLimitTask(1*time.Second, newTask(5))
	require.Error(t, result.Error())
	require.True(t, result.Timeout())
	require.Nil(t, result.Value())

	//任务超时，后台任务仍在执行，因为没有办法中断已经启动的协程
	time.Sleep(5 * time.Second)    //等待后台任务执行完成
	require.Nil(t, result.Value()) //任务已经超时，所以任务执行完成，其返回值也不会再设置给result
}

func TestRunUtilCancel(t *testing.T) {
	//async.RunUtilCancel()内部使用for-select执行任务
	//在外部传入的ctx.Done()或任务内部调用c.Abort()时结束任务(退出循环)

	//以下任务耗时4秒，任务在第3次循环时才能判断外部ctx取消
	//如果将超时时间设置为2秒，则任务可能耗时2秒/4秒，即任务可能多执行1次
	start := time.Now()
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	<-async.RunUtilCancel(ctx, func(c async.Context) {
		if c.Done() {
			fmt.Println("task1 done", time.Now().Second())
			return
		}

		fmt.Println("task1 do: ", time.Now().Second())
		time.Sleep(2 * time.Second)
	})
	require.WithinDuration(t, time.Now(), start.Add(4*time.Second), 1*time.Second)

	//以下任务耗时3秒，即使外部ctx设置为1秒后超时
	//因为第1次循环时，任务sleep()，下次循环时才能检查ctx取消
	start = time.Now()
	ctx, _ = context.WithTimeout(context.Background(), 1*time.Second)
	<-async.RunUtilCancel(ctx, func(c async.Context) {
		if c.Done() {
			fmt.Println("task2 done", time.Now().Second())
			return
		}

		fmt.Println("task2 start", time.Now().Second())
		time.Sleep(3 * time.Second)
	})
	require.WithinDuration(t, time.Now(), start.Add(3*time.Second), 1*time.Second)

	//执行3次后主动取消执行任务
	count := int64(0)
	<-async.RunUtilCancel(context.Background(), func(c async.Context) {
		count = c.Count()

		if c.Done() {
			//主动取消，不会执行以下代码
			fmt.Println("task3 done")
			return
		}

		//c.Count()从1开始计数
		fmt.Println("task3 do: ", c.Count())

		if c.Count() == 3 {
			c.Abort() //主动取消
		}
	})
	require.EqualValues(t, 3, count)
}
