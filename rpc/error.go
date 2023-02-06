package rpc

import (
	"fmt"
	"github.com/bingooh/b-go-util/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
)

var codeRpcCodeMap = map[int]codes.Code{
	util.ErrCodeIllegalArg:   codes.InvalidArgument,
	util.ErrCodeIllegalState: codes.FailedPrecondition,
	util.ErrCodeUnAuth:       codes.Unauthenticated,
	util.ErrCodeForbidden:    codes.PermissionDenied,
	util.ErrCodeTooOften:     codes.ResourceExhausted,
	util.ErrCodeTimeout:      codes.DeadlineExceeded,
	util.ErrCodeInternal:     codes.Internal,
	util.ErrCodeNotFound:     codes.NotFound,
	util.ErrCodeCanceled:     codes.Canceled,
	util.ErrCodeAborted:      codes.Aborted,
	util.ErrCodeUnknown:      codes.Unknown,
	util.ErrCodeOK:           codes.OK,
}

var rpcCodeCodeMap = func() map[codes.Code]int {
	m := make(map[codes.Code]int, len(codeRpcCodeMap))
	for k, v := range codeRpcCodeMap {
		m[v] = k
	}

	return m
}()

func ToRpcErrCode(bizErrCode int, defaultRpcErrCode codes.Code) codes.Code {
	if v, ok := codeRpcCodeMap[bizErrCode]; ok {
		return v
	}

	return defaultRpcErrCode
}

func ToBizErrCode(code codes.Code, defaultBizErrCode int) int {
	if v, ok := rpcCodeCodeMap[code]; ok {
		return v
	}

	return defaultBizErrCode
}

func IsRpcErrCode(code codes.Code) bool {
	return code >= codes.OK && code <= codes.Unauthenticated //todo 依赖库更新应同步修改此方法
}

func IsRpcErr(err error) bool {
	_, ok := status.FromError(err) //err==nil将返回true
	return ok
}

func ToRpcErr(err error, args ...interface{}) error {
	if IsRpcErr(err) {
		return err
	}

	//可以考虑将错误对象放入错误详情，参考：status.WithDetails()/errdetails.ErrorInfo
	code := codes.Unknown
	if e, ok := util.AsBizError(err); ok {
		code = ToRpcErrCode(e.Code(), codes.Code(e.Code()))
	}

	if len(args) == 0 {
		return status.Error(code, err.Error())
	}

	args = append([]interface{}{err}, args...)
	return status.Errorf(code, util.Sprintf(args...))
}

func ToBizErr(err error) error {
	if err == nil {
		return err
	}

	if s, ok := status.FromError(err); ok && s != nil {
		code := ToBizErrCode(s.Code(), int(s.Code()))
		//去掉错误码，避免重复，此方法需要与BizError格式化错误消息方法同步
		msg := strings.TrimPrefix(s.Message(), fmt.Sprintf(`(%v)`, code))
		return util.NewBizError(code, msg)
	}

	return util.ToBizError(err)
}
