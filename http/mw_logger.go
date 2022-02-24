package http

import (
	"b-go-util/slog"
	"github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"time"
)

func newLogger(tag string) *zap.Logger {
	return slog.NewLogger(`http`, tag)
}

//创建用于输出gin日志的日志器，默认读取conf/log_http配置文件，如无则调用slog.NewLogger()
func newHttpLogger() *zap.Logger {
	if logger, err := slog.NewLoggerFromCfgFile(`log_http`); err == nil {
		return logger.With(slog.NewTagField(`http`))
	}

	return slog.NewLogger(`http`)
}

func MWLogger() gin.HandlerFunc {
	return ginzap.Ginzap(newHttpLogger(), time.RFC3339, false)
}

//输出崩溃错误日志，不能同时使用中间件gin.Recovery()
func MWPanicLogger() gin.HandlerFunc {
	return ginzap.RecoveryWithZap(newHttpLogger(), true)
}
