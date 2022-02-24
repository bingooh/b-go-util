package util

import (
	"errors"
	"github.com/bingooh/b-go-util/util"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestRetryCounter(t *testing.T) {
	r := require.New(t)

	//固定间隔时长重试计数器
	maxCount := 3 //最大重试次数
	c1 := util.NewRetryCounter(maxCount, 1*time.Second)
	r.EqualValues(0, c1.Count()) //当前重试次数为0
	for i := 1; i <= maxCount+3; i++ {
		v := c1.NextInterval() //调用此方法将导致重试次数+1

		if i <= maxCount {
			r.EqualValues(i, c1.Count())
			r.EqualValues(1*time.Second, v)
		} else {
			r.EqualValues(maxCount, c1.Count())
			r.EqualValues(0, v)
		}
	}

	//步进间隔时长重试计数器，每次递增2秒
	c2 := util.NewStepRetryCounter(maxCount, 0, 2*time.Second, 0)
	for i := 1; i <= maxCount+3; i++ {
		v := c2.NextInterval()

		if i <= maxCount {
			r.EqualValues(i, c2.Count())
			r.EqualValues(time.Duration(i)*2*time.Second, v)
		} else {
			r.EqualValues(maxCount, c2.Count())
			r.EqualValues(0, v)
		}
	}

	//步进间隔时长重试计数器，每次递增2秒，最大间隔9秒，不限最大重试次数
	c3 := util.NewStepRetryCounter(0, 0, 2*time.Second, 9*time.Second)
	for i := 1; i <= 10; i++ {
		v := c3.NextInterval()
		r.EqualValues(i, c3.Count())

		if i <= 4 {
			r.EqualValues(time.Duration(i)*2*time.Second, v)
		} else {
			r.EqualValues(9*time.Second, v)
		}
	}

	//步进间隔时长重试计数器，每次递增1秒，最大间隔3秒，最大重试次数8。第5次时重试成功退出执行
	//c4 := util.NewStepRetryCounter(8, 0, 1*time.Second, 3*time.Second)
	c4 := util.NewRetryCounter(8, 1*time.Second)
	err := util.DoRetry(c4, func() error {
		util.Log(`count:%v`, c4.Count())

		if c4.Count() == 5 {
			return nil
		}

		return errors.New(`not done`)
	})

	r.NoError(err)

}
