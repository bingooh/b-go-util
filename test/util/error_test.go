package util

import (
	"errors"
	"fmt"
	"github.com/bingooh/b-go-util/util"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestBizError(t *testing.T) {
	r := require.New(t)

	//如果err为空，则错误码为ErrCodeOK
	var e1 error
	var e2 *util.BizError
	r.True(util.IsOKErr(e1))
	r.True(util.IsOKErr(e2))

	e11, ok := util.AsBizError(e1)
	r.True(e11 == nil)
	r.False(ok)

	//e2为nil *BizError，转换结果nil,true
	e22, ok := util.AsBizError(e2)
	r.True(e22 == nil)
	r.True(ok)

	e3 := errors.New(`e3`)
	e33, ok := util.AsBizError(e3)
	r.True(e33 == nil)
	r.Equal(ok, util.IsBizError(e3))

	//ToBizError()不会返回nil err
	e333 := util.ToBizError(e3)
	r.True(e333 != nil)
	r.Equal(util.ErrCodeUnknown, e333.Code())

	e111 := util.ToBizError(e1) //e1为nil
	r.True(e111 != nil)
	r.True(util.IsOKErr(e111))

	//e5->e4
	e4 := errors.New(`e4`)
	e5 := util.NewNilError(e4, `e5`)
	r.Equal(e4, e5.Unwrap())
	r.True(errors.Is(e5, e4))

	//e6->e5->e4
	e6 := fmt.Errorf(`e6->%w`, e5)
	r.True(errors.Is(e6, e5))
	r.True(errors.Is(e6, e4))

	//因为e6.cause==e5,所以以下测试通过
	r.True(util.IsNilErr(e6))
	r.True(util.IsBizError(e6))

	_, ok = e6.(*util.BizError)
	r.False(ok) //e6本身不是*BizError类型

	e55, ok := util.AsBizError(e6)
	r.True(ok)
	r.True(util.IsNilErr(e55))
	r.True(e5 == e55) //指针地址相同，指向同1个对象

	//输出格式为:(code)msg->cause
	fmt.Println(e6)

	//自定义错误码建议3位整数开头，args参加见NewBizError()说明
	e7 := util.NewBizError(100, e6, `e7[%v]`, time.Now().Unix())
	fmt.Println(e7)
}
