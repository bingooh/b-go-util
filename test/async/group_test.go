package async

import (
	"context"
	"errors"
	"fmt"
	"github.com/bingooh/b-go-util/async"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

var assertFailErr = errors.New(`assert fail`)

func TestGroup(t *testing.T) {
	//Group用于执行1组任务并保存任务的执行结果
	g := async.NewGroup()

	//添加5个任务，分别沉睡1-5秒后结束
	for i := 1; i <= 5; i++ {
		g.RunTask(newTask(i))
	}

	//添加1个报错任务，沉睡100纳秒后结束，此任务最先完成
	g.RunTask(async.ToErrTask(func() error {
		fmt.Println("err task start")
		time.Sleep(100 * time.Nanosecond)
		fmt.Println("err task done")
		return assertFailErr
	}))

	//等待全部任务执行完成并获取任务组结果
	//不建议Wait()后再调用Run()添加新任务
	result := g.Wait()
	require.Equal(t, 6, result.Size())         //总共6个任务，
	require.Equal(t, 5, result.FirstDoneIdx()) //报错任务最后添加，但最先执行完成
	require.Equal(t, 0, result.FirstOkIdx())   //第1个成功执行的任务

	taskResult := result.Get(result.FirstDoneIdx())      //获取错误任务执行结果
	require.Equal(t, result.Error(), taskResult.Error()) //group的错误仅保留第1个错误任务返回的错误
	require.Equal(t, assertFailErr, taskResult.Error())
	require.True(t, result.HasError())

	//再次调用Wait()获取相同结果，并转换为ResultMap。key为添加任务时的顺序号(0开始)
	g.Wait().ResultMap().Each(func(key int, r async.Result) {
		fmt.Printf("task %v result: %v \n", key, r.Value())
	})

	//再次调用Wait()获取相同结果，并转换为ResultList， 任务按照的其添加顺序排序
	g.Wait().ResultList().Each(func(i int, r async.Result) {
		fmt.Printf("task %v result: %v \n", i, r.Value())
	})
}

func TestGroupWait(t *testing.T) {
	g := async.NewGroup()

	//添加3个3秒内完成的任务
	for i := 1; i <= 3; i++ {
		g.RunTask(newTask(i))
	}

	//添加3个6秒后才能完成的任务
	for i := 6; i <= 8; i++ {
		g.RunTask(newTask(i))
	}

	require.EqualValues(t, 6, g.ExistTaskCount()) //全部添加6个任务

	//等待4秒后超时退出，此时group result仅包含前3个任务执行结果
	result := g.WaitOrTimeout(4 * time.Second)
	require.Equal(t, 3, result.Size())
	require.True(t, result.HasError())
	require.True(t, result.Timeout())
	require.Equal(t, context.DeadlineExceeded, result.Error()) //result错误保存等待超时错误或者第1个报错任务返回的错误
	require.EqualValues(t, 3, g.DoneTaskCount())               //3个已完成任务
	require.EqualValues(t, 3, g.PendingTaskCount())            //3个待完成任务

	//再次等待5秒后直到所有任务结束，此时group result仍然只包含前3个任务执行结果
	//即：首次等待超时后，后续执行完成的任务结果将被直接忽略，不会添加到当前group里
	//但是group.DoneTaskCount()仍然包含后继续执行完成的任务数
	result = g.WaitOrTimeout(5 * time.Second)
	require.Equal(t, 3, result.Size())
	require.EqualValues(t, 6, g.DoneTaskCount())    //6个任务全部完成
	require.EqualValues(t, 0, g.PendingTaskCount()) //没有待完成的任务
}

func TestGroupRunTimeLimitTask(t *testing.T) {
	g := async.NewGroup()

	//偶数任务1秒超时,奇数任务10秒超时，每个任务沉睡2-5秒后结束
	//即偶数任务都将返回超时错误，而奇数任务成功执行
	for i := 2; i <= 5; i++ {
		n := 10
		if i%2 == 0 {
			n = 1
		}

		timeout := time.Duration(n) * time.Second
		g.RunTimeLimitTask(timeout, newTask(i))
	}

	//遍历每个任务执行结果
	g.Wait().ResultList().Each(func(i int, r async.Result) {
		fmt.Printf("task %v  timeout: %v result: %v \n", i, r.Timeout(), r.Value())

		if i%2 == 0 {
			//偶数任务超时，且返回结果值为nil
			require.True(t, r.Timeout() && r.Value() == nil)
		} else {
			require.True(t, !r.Timeout() && r.Value() != nil)
		}
	})
}

func TestGroupWithPool(t *testing.T) {
	//使用pool限制group执行任务的协程数，以下创建仅有1个协程的协程池
	pool := async.NewWorkerPool(1, 1, 0)
	defer pool.Close()

	//创建任务组并设置使用的协程池
	g := async.NewGroup().WithPool(pool)

	//添加3个任务，每个任务沉睡1秒结束。总共耗时3*1秒
	start := time.Now()
	for i := 0; i < 3; i++ {
		g.RunTask(newTask(1))
	}
	g.Wait()

	require.WithinDuration(t, time.Now(), start.Add(3*time.Second), 1*time.Second)
}
