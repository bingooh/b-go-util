package http

import (
	"github.com/bingooh/b-go-util/util"
	"github.com/gin-gonic/gin"
	"net/http"
)

type Body struct {
	Data  interface{} `json:"data,omitempty"`  //数据
	Error interface{} `json:"error,omitempty"` //错误
	Total int64       `json:"total,omitempty"` //总数
}

func (b Body) ToMap() map[string]interface{} {
	m := make(map[string]interface{})
	if b.Data != nil {
		m[`data`] = b.Data
	}

	if b.Error != nil {
		m[`error`] = b.Error
	}

	if b.Total != 0 {
		m[`total`] = b.Total
	}

	return m
}

func NoRouteHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		err := NewError(http.StatusNotFound, util.ErrCodeUnknown, `页面未找到`)
		c.JSON(err.Status(), err)
	}
}
