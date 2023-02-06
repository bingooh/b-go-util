package rpc

import (
	"errors"
	"fmt"
	"github.com/bingooh/b-go-util/rpc"
	"github.com/bingooh/b-go-util/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"testing"
)

func TestRpcErr(t *testing.T) {
	r := require.New(t)

	toBizErr := func(err error) *util.BizError {
		if err == nil {
			return nil
		}

		return rpc.ToBizErr(err).(*util.BizError)
	}

	e1 := errors.New(`e1`)
	e11 := rpc.ToRpcErr(e1)
	s11, ok := status.FromError(e11)
	r.True(ok)
	r.EqualValues(codes.Unknown, s11.Code())

	e111 := toBizErr(s11.Err())
	r.EqualValues(util.ErrCodeUnknown, e111.Code())

	e2 := util.NewIllegalArgError(`e2`)
	e22 := rpc.ToRpcErr(e2)
	s22, ok := status.FromError(e22)
	r.True(ok)
	r.EqualValues(rpc.ToRpcErrCode(e2.Code(), codes.Unknown), s22.Code())
	r.True(rpc.IsRpcErrCode(s22.Code()))

	e222 := toBizErr(s22.Err())
	r.EqualValues(rpc.ToBizErrCode(s22.Code(), util.ErrCodeUnknown), e222.Code())

	//自定义错误码
	e3 := util.NewBizError(9999, `e3`)
	e33 := rpc.ToRpcErr(e3)
	s33, ok := status.FromError(e33)
	r.True(ok)
	r.EqualValues(e3.Code(), s33.Code())
	r.False(rpc.IsRpcErrCode(s33.Code())) //非rpc err code
	fmt.Println(e33)

	e333 := toBizErr(s33.Err())
	r.EqualValues(s33.Code(), e333.Code())

	var e4 error
	e44 := rpc.ToRpcErr(e4)
	r.Nil(e44)
	r.True(rpc.IsRpcErr(e4)) //nil err都为rpc err

	e444 := toBizErr(e44)
	r.Nil(e444) //返回nil，util.ToBizErr()返回非空

	e5 := status.Error(codes.OK, `e5`)
	e55 := rpc.ToRpcErr(e5)
	r.Equal(e5, e55) //直接返回

	e6 := errors.New(`e6`)
	fmt.Println(rpc.ToRpcErr(e6, `my`))

	e7 := util.NewIllegalArgError(`e7`)
	fmt.Println(rpc.ToRpcErr(e7, `my`))
}
