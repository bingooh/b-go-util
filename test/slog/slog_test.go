package slog

import (
	"errors"
	"github.com/bingooh/b-go-util/slog"
	"go.uber.org/zap"
	"testing"
)

func TestSlog(t *testing.T) {
	slog.MustInitDefaultRootLogger()
	//slog.MustInitDefaultRootLoggerWithLevel(zapcore.WarnLevel)

	logger := slog.NewLogger(`test`)
	logger.Debug(`debug`)
	logger.Info(`info`)
	logger.Warn(`warn`)
	logger.Error(`err`, zap.Error(errors.New(`this is err`)))

	slog.SetLogTagFilter(func(tag string) bool {
		return tag == `tt.1`
	})

	//以下日志将被过滤不显示
	logger1 := slog.NewLogger(`tt`, `1`)
	logger1.Info(`hello`)
}
