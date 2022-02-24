package http

import (
	"github.com/bingooh/b-go-util/_string"
	"github.com/bingooh/b-go-util/util"
	"github.com/gin-gonic/gin"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
	"net/http"
	"runtime/debug"
	"strconv"
)

type MWErrorHandler struct {
	logger                           *zap.Logger
	toHttpErrHook                    MWErrorHandlerToHttpErrHook
	sendRspHook                      MWErrorHandlerSendRspHook
	rspErrField                      string //错误对象对应的响应字段名称
	enableLog400Err                  bool   //是否输出400错误的日志，默认false
	enableParseHttpStatusFromErrCode bool   //是否从错误码里解析http响应状态码，默认true。注意：仅解析至少为4位数的错误码
}

//回调函数，解析错误对象为http.Error
type MWErrorHandlerToHttpErrHook func(ctx *gin.Context, handler *MWErrorHandler) *Error

//回调函数，发送错误响应
type MWErrorHandlerSendRspHook func(ctx *gin.Context, handler *MWErrorHandler, err *Error)

func NewMWErrorHandler() *MWErrorHandler {
	return &MWErrorHandler{
		logger:                           newLogger(`MWErrorHandler`),
		enableParseHttpStatusFromErrCode: true,
	}
}

func (h *MWErrorHandler) EnableLog400Err(enable bool) *MWErrorHandler {
	h.enableLog400Err = enable
	return h
}

func (h *MWErrorHandler) EnableParseHttpStatusFromErrCode(enable bool) *MWErrorHandler {
	h.enableParseHttpStatusFromErrCode = enable
	return h
}

func (h *MWErrorHandler) WithRspErrField(field string) *MWErrorHandler {
	h.rspErrField = field
	return h
}

func (h *MWErrorHandler) WithSendRspHook(fn MWErrorHandlerSendRspHook) *MWErrorHandler {
	h.sendRspHook = fn
	return h
}

func (h *MWErrorHandler) WithToHttpErrHook(fn MWErrorHandlerToHttpErrHook) *MWErrorHandler {
	h.toHttpErrHook = fn
	return h
}

func (h *MWErrorHandler) sendErrRsp(c *gin.Context, httpErr *Error, isPanicErr bool) {
	if httpErr == nil {
		return
	}

	if h.enableLog400Err || httpErr.Status() > http.StatusBadRequest {
		fields := []zap.Field{
			zap.Int(`code`, httpErr.Code()), zap.Int(`status`, httpErr.Status()),
			zap.String(`method`, c.Request.Method), zap.String(`url`, c.Request.URL.String()), zap.Error(httpErr),
		}

		if isPanicErr {
			fields = append(fields, zap.ByteString(`stack`, debug.Stack()))
		}

		h.logger.Error(`http请求出错`, fields...)
	}

	if h.sendRspHook != nil {
		h.sendRspHook(c, h, httpErr)
		return
	}

	if _string.Empty(h.rspErrField) {
		c.JSON(httpErr.status, httpErr)
		return
	}

	c.JSON(httpErr.status, gin.H{h.rspErrField: httpErr})
}

//处理请求
func (h *MWErrorHandler) Handle(c *gin.Context) {
	defer util.OnExit(func(err error) {
		if err != nil {
			h.sendErrRsp(c, New500Error(util.GetBizErrCode(err), err, `服务器崩溃`), true)
			return
		}

		if !c.IsAborted() && len(c.Errors) > 0 {
			//如果handler调用过c.Abort()，即handler自行发送响应，则不处理错误
			h.sendErrRsp(c, h.ToHttpErr(c), false)
		}
	})

	c.Next()
}

//是否为验证错误
func (h *MWErrorHandler) IsValidationErr(err error) bool {
	if e, ok := err.(*gin.Error); ok {
		if e.Type == gin.ErrorTypeBind {
			//调用c.MustBind()会返回此错误类型
			return true
		}

		err = e.Err //获取cause
	}

	switch err.(type) {
	case validator.ValidationErrors, validation.Errors, validation.Error:
		return true
	default:
		return false
	}
}

func (h *MWErrorHandler) ToHttpErrCode(err error) int {
	if e, ok := err.(*gin.Error); ok {
		err = e.Err
	}

	if es, ok := err.(validation.Errors); ok && len(es) > 0 {
		//只取第1个错误
		for _, v := range es {
			err = v
			break
		}
	}

	if e, ok := err.(validation.Error); ok {
		if code, ee := strconv.Atoi(e.Code()); ee == nil {
			return code
		}
	}

	return util.GetBizErrCode(err)
}

func (h *MWErrorHandler) toHttpErrStatus(code int) int {
	if !h.enableParseHttpStatusFromErrCode || code <= 999 {
		return util.ToHttpStatus(code)
	}

	v := strconv.Itoa(code)[:3]
	s, _ := strconv.Atoi(v)
	return s
}

func (h *MWErrorHandler) ToHttpErr(c *gin.Context) *Error {
	if len(c.Errors) == 0 {
		return nil
	}

	if h.toHttpErrHook != nil {
		return h.toHttpErrHook(c, h)
	}

	err := c.Errors.Last()
	if e, ok := AsError(err); ok {
		return e
	}

	code := h.ToHttpErrCode(err)
	if h.IsValidationErr(err) {
		return New400Error(code, err.Error())
	}

	return NewError(h.toHttpErrStatus(code), code, err.Error())
}
