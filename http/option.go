package http

import (
	"github.com/bingooh/b-go-util/_string"
	"github.com/bingooh/b-go-util/util"
	"github.com/gin-gonic/gin"
)

type ServerOption struct {
	//以下为Gin的全局配置
	GinMode string //默认为gin.DebugMode

	ListenAddress string //服务器监听地址
}

func (o *ServerOption) MustNormalize() *ServerOption {
	util.AssertOk(o != nil, "option为空")

	if _string.Empty(o.GinMode) {
		o.GinMode = gin.DebugMode
	}

	return o
}

// 重置Gin全局配置
func (o *ServerOption) ResetGinGlobalCfg() {
	gin.SetMode(o.GinMode)
}
