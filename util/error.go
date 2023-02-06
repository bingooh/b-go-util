package util

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrBreak    = errors.New(`break`)
	ErrContinue = errors.New(`continue`)
	ErrReturn   = errors.New(`return`)
)

// 错误码，建议自定义业务错误码使用6位数，前3位可表示http响应状态码
const (
	ErrCodeUnknown      int = -1
	ErrCodeOK           int = 0
	ErrCodeNil          int = 1
	ErrCodeInternal     int = 2
	ErrCodeAssertFail   int = 3
	ErrCodeIllegalArg   int = 4
	ErrCodeIllegalState int = 5
	ErrCodeTypeCast     int = 6
	ErrCodeUnAuth       int = 7
	ErrCodeForbidden    int = 8
	ErrCodeTimeout      int = 9
	ErrCodeTooOften     int = 10
	ErrCodeCanceled     int = 11
	ErrCodeAborted      int = 12
	ErrCodeNotFound     int = 13
	ErrCodeRedis        int = 100
	ErrCodeDB           int = 200
)

func NewNilError(args ...interface{}) *BizError {
	return NewBizError(ErrCodeNil, args...)
}

func NewNotFoundError(args ...interface{}) *BizError {
	return NewBizError(ErrCodeNotFound, args...)
}

func NewAssertFailError(args ...interface{}) *BizError {
	return NewBizError(ErrCodeAssertFail, args...)
}

func NewInternalError(args ...interface{}) *BizError {
	return NewBizError(ErrCodeInternal, args...)
}

func NewIllegalArgError(args ...interface{}) *BizError {
	return NewBizError(ErrCodeIllegalArg, args...)
}

func NewIllegalStateError(args ...interface{}) *BizError {
	return NewBizError(ErrCodeIllegalState, args...)
}

func NewTypeCastError(args ...interface{}) *BizError {
	return NewBizError(ErrCodeTypeCast, args...)
}

func NewUnAuthError(args ...interface{}) *BizError {
	return NewBizError(ErrCodeUnAuth, args...)
}

func NewForbiddenError(args ...interface{}) *BizError {
	return NewBizError(ErrCodeForbidden, args...)
}

func NewTooOftenError(args ...interface{}) *BizError {
	return NewBizError(ErrCodeTooOften, args...)
}

func NewRedisError(args ...interface{}) *BizError {
	return NewBizError(ErrCodeRedis, args...)
}

func NewDBError(args ...interface{}) *BizError {
	return NewBizError(ErrCodeDB, args...)
}

type BizError struct {
	code  int
	msg   string
	cause error
}

// NewBizError args参数格式：err / format,args... / err,format,args...
func NewBizError(code int, args ...interface{}) *BizError {
	var cause error
	if n := len(args); n > 0 {
		if err, ok := args[0].(error); ok {
			cause = err
			args = args[1:]
		}
	}

	msg := fmt.Sprintf(`(%v)%v`, code, Sprintf(args...))
	if cause != nil {
		msg = fmt.Sprintf(`%v->%v`, msg, cause)
	}

	return &BizError{code: code, cause: cause, msg: msg}
}

func (e *BizError) Error() string {
	if e == nil {
		return ``
	}

	return e.msg
}

func (e *BizError) Code() int {
	if e == nil {
		return ErrCodeOK
	}

	return e.code
}

func (e *BizError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.cause
}

// AsBizError 注：如果err为nil *BizError，则返回值nil,true
func AsBizError(err error) (*BizError, bool) {
	var e *BizError

	ok := errors.As(err, &e)
	return e, ok
}

func ToBizError(err error) *BizError {
	if e, ok := AsBizError(err); ok && e != nil {
		return e
	}

	if err == nil {
		return NewBizError(ErrCodeOK)
	}

	return NewBizError(ErrCodeUnknown, err.Error())
}

func IsBizError(err error) bool {
	_, ok := AsBizError(err)
	return ok
}

func GetBizErrCode(err error) int {
	if err == nil {
		return ErrCodeOK
	}

	if e, ok := AsBizError(err); ok {
		return e.Code()
	}

	return ErrCodeUnknown
}

func HasErrCode(err error, code int) bool {
	if err == nil {
		return code == ErrCodeOK
	}

	if e, ok := AsBizError(err); ok {
		return e.Code() == code
	}

	return false
}

func IsOKErr(err error) bool {
	return HasErrCode(err, ErrCodeOK)
}

func IsNilErr(err error) bool {
	return HasErrCode(err, ErrCodeNil)
}

func IsUnAuthErr(err error) bool {
	return HasErrCode(err, ErrCodeUnAuth)
}

func IsForbiddenErr(err error) bool {
	return HasErrCode(err, ErrCodeForbidden)
}

// http.StatusPreconditionFailed适用于http协议，用于服务端检查是否满足http头请求参数。不适合业务错误
var codeHttpStatusMap = map[int]int{
	ErrCodeIllegalArg:   http.StatusBadRequest,
	ErrCodeIllegalState: http.StatusTooEarly,
	ErrCodeUnAuth:       http.StatusUnauthorized,
	ErrCodeForbidden:    http.StatusForbidden,
	ErrCodeTooOften:     http.StatusTooManyRequests,
	ErrCodeNotFound:     http.StatusNotFound,
	ErrCodeOK:           http.StatusOK,
}

func ToHttpStatus(bizErrCode int, defaultHttpStatus int) int {
	if v, ok := codeHttpStatusMap[bizErrCode]; ok {
		return v
	}
	return defaultHttpStatus
}
