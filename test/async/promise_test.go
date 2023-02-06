package async

import (
	"context"
	"errors"
	"fmt"
	"github.com/bingooh/b-go-util/async"
	"github.com/bingooh/b-go-util/util"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// 创建延时任务，沉睡i秒后任务结束并返回cause
func newErrJob(v *util.AtomicInt64, i int, cause error) func() error {
	return func() error {
		job(i)

		if cause == nil {
			v.Set(int64(i))
		}

		return cause
	}
}

func TestDoAll(t *testing.T) {
	r := require.New(t)
	e1 := errors.New(`e1`)
	e2 := errors.New(`e2`)
	e3 := errors.New(`e3`)

	//DoAll()串行执行任务直到遇到第1个错误并返回
	//执行3个任务，job2返回错误导致提前结束执行
	v1 := util.NewAtomicInt64(0)
	err := async.DoAll(
		newErrJob(v1, 1, nil),
		newErrJob(v1, 2, e2),
		newErrJob(v1, 3, nil),
	)
	r.Equal(e2, err)
	r.EqualValues(1, v1.Value())

	fmt.Println(`--------------`)

	//执行2个任务，ctx取消导致job3未执行
	v2 := util.NewAtomicInt64(0)
	ctx1, _ := context.WithTimeout(context.TODO(), 2*time.Second)
	err = async.DoCancelableAll(
		ctx1,
		newErrJob(v2, 1, nil),
		newErrJob(v2, 3, nil),
	)
	r.Equal(context.DeadlineExceeded, err)
	r.EqualValues(1, v2.Value())

	fmt.Println(`--------------`)

	//DoAny()串行执行任务直到第1个执行成功，如果全部失败则返回最后1个错误
	v3 := util.NewAtomicInt64(0)
	err = async.RunAny(
		newErrJob(v3, 1, e1),
		newErrJob(v3, 2, nil),
		newErrJob(v3, 3, e3),
	)
	r.NoError(err)
	r.EqualValues(2, v3.Value())

	fmt.Println(`--------------`)

	//执行2个任务，ctx取消导致job3未执行
	v4 := util.NewAtomicInt64(0)
	ctx2, _ := context.WithTimeout(context.TODO(), 2*time.Second)
	err = async.DoCancelableAny(
		ctx2,
		newErrJob(v4, 1, e1),
		newErrJob(v4, 3, nil),
	)
	r.Equal(context.DeadlineExceeded, err)
	r.EqualValues(0, v4.Value())
}

func TestRunAll(t *testing.T) {
	r := require.New(t)
	e1 := errors.New(`e1`)
	e2 := errors.New(`e2`)
	e3 := errors.New(`e3`)

	//RunAll()并发执行任务直到遇到第1个错误并返回
	//执行3个任务，job2返回错误导致提前结束执行
	v1 := util.NewAtomicInt64(0)
	err := async.RunAll(
		newErrJob(v1, 1, nil),
		newErrJob(v1, 2, e2),
		newErrJob(v1, 3, nil),
	)
	r.Equal(e2, err)
	r.EqualValues(1, v1.Value())

	//RunAll()结束执行并不会结束执行job，除非job本身支持取消执行(因为无法中断协程）
	fmt.Println(`run all done`)
	time.Sleep(3 * time.Second)  //等待job3执行完成
	r.EqualValues(3, v1.Value()) //结果被job3修改为3

	fmt.Println(`--------------`)

	//执行2个任务，ctx取消导致job3未执行
	v2 := util.NewAtomicInt64(0)
	ctx1, _ := context.WithTimeout(context.TODO(), 2*time.Second)
	err = async.RunCancelableAll(
		ctx1,
		newErrJob(v2, 1, nil),
		newErrJob(v2, 3, nil),
	)
	r.Equal(context.DeadlineExceeded, err)
	r.EqualValues(1, v2.Value())

	fmt.Println(`--------------`)

	//RunAny()并发执行任务直到第1个执行成功，如果全部失败则返回最后1个错误
	v3 := util.NewAtomicInt64(0)
	err = async.RunAny(
		newErrJob(v3, 1, e1),
		newErrJob(v3, 2, nil),
		newErrJob(v3, 3, e3),
	)
	r.NoError(err)
	r.EqualValues(2, v3.Value())

	fmt.Println(`--------------`)

	//执行2个任务，ctx取消导致job3未执行
	v4 := util.NewAtomicInt64(0)
	ctx2, _ := context.WithTimeout(context.TODO(), 2*time.Second)
	err = async.RunCancelableAny(
		ctx2,
		newErrJob(v4, 1, e1),
		newErrJob(v4, 3, nil),
	)
	r.Equal(context.DeadlineExceeded, err)
	r.EqualValues(0, v4.Value())
}

func TestRunAllCollectValue(t *testing.T) {
	r := require.New(t)
	e1 := errors.New(`e1`)

	//每个任务只修改自己的变量，无需同步
	v1 := 0
	v2 := 0
	async.RunAll(
		func() error {
			v1 = 1
			return nil
		},
		func() error {
			v2 = 2
			return nil
		},
	)
	r.Equal(1, v1)
	r.Equal(2, v2)

	//读写同1个变量,需要同步
	v3 := util.NewAtomicInt64(0)
	async.RunAll(
		func() error {
			v3.Incr(1)
			return nil
		},
		func() error {
			v3.Incr(1)
			return nil
		},
	)
	r.EqualValues(2, v3.Value())

	//并发执行，不能保证任务执行顺序，导致结果不确定
	v4 := 0
	err := async.RunAll(
		func() error {
			v4 = 1
			return nil
		},
		func() error {
			return e1
		},
	)
	r.Equal(e1, err)
	r.True(v4 == 0 || v4 == 1) //v4最后会被设置为1，因为没法终止执行协程

	//每个任务的结果保存到map
	rs := async.NewSyncResultMap()
	async.RunAll(
		func() error {
			rs.Put(1, async.NewResult(1, nil))
			return nil
		},
		func() error {
			rs.Put(2, async.NewResult(2, nil))
			return nil
		},
	)

	r.Equal(1, rs.Get(1).MustInt())
	r.Equal(2, rs.Get(2).MustInt())

	//每个任务的结果保存到map，不管任务是否执行成功
	rs = async.NewSyncResultMap()
	g := async.NewWaitGroup()
	g.Run(func() {
		rs.Put(0, async.NewResult(0, e1))
	})
	g.Run(func() {
		rs.Put(1, async.NewResult(1, nil))
	})
	g.Wait()

	r.Equal(2, rs.Size())
	r.Equal(e1, rs.Get(0).Error())
	r.Equal(1, rs.Get(1).MustInt())
	r.False(rs.Has(2))
	r.Nil(rs.Get(3))

}
