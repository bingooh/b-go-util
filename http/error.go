package http

import (
	"errors"
	"fmt"
	"github.com/bingooh/b-go-util/util"
	"net/http"
)

const (
	ErrCodeOK = 0
)

type Error struct {
	status int
	code   int
	msg    string
	cause  error
}

// 用于序列化json
type ErrorAlias struct {
	Status int    `json:"status"` //状态码
	Code   int    `json:"code"`   //错误码
	Msg    string `json:"msg"`    //错误消息
}

// args参数格式：err / format,args... / err,format,args...
func NewError(status, code int, args ...interface{}) *Error {
	var cause error
	if n := len(args); n > 0 {
		if err, ok := args[0].(error); ok {
			cause = err
			args = args[1:]
		}
	}

	//msg := fmt.Sprintf(`[status=%v,code=%v]%v`, status, code, util.Sprintf(args...))
	msg := util.Sprintf(args...)
	if cause != nil {
		msg = fmt.Sprintf(`%v->%v`, msg, cause)
	}

	return &Error{status: status, code: code, cause: cause, msg: msg}
}

func New400Error(code int, args ...interface{}) *Error {
	return NewError(http.StatusBadRequest, code, args...)
}

func New500Error(code int, args ...interface{}) *Error {
	return NewError(http.StatusInternalServerError, code, args...)
}

func (e *Error) Error() string {
	if e == nil {
		return ``
	}

	return e.msg
}

func (e *Error) Status() int {
	if e == nil {
		return http.StatusOK
	}

	return e.status
}

func (e *Error) Code() int {
	if e == nil {
		return ErrCodeOK
	}

	return e.code
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.cause
}

func (e *Error) OK() bool {
	return e == nil || e.status == http.StatusOK
}

func (e *Error) MarshalJSON() ([]byte, error) {
	if e == nil {
		return util.MarshalJSON(nil)
	}

	return util.MarshalJSON(&ErrorAlias{
		Status: e.status,
		Code:   e.code,
		Msg:    e.msg,
	})
}

func (e *Error) UnmarshalJSON(data []byte) (err error) {
	a := &ErrorAlias{}
	if err = util.UnmarshalJSON(data, a); err == nil {
		e.status = a.Status
		e.code = a.Code
		e.msg = a.Msg
	}

	return
}

// 注：如果err为nil *Error，则返回值nil,true
func AsError(err error) (*Error, bool) {
	var e *Error

	ok := errors.As(err, &e)
	return e, ok
}

func IsError(err error) bool {
	_, ok := AsError(err)
	return ok
}
