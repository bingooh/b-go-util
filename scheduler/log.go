package scheduler

import (
	"fmt"
	"github.com/bingooh/b-go-util/slog"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"strings"
	"time"
)

//创建用于输出db日志的日志器，默认读取conf/log_db配置文件，如无则调用slog.NewLogger()
func newSchedulerZapLogger() *zap.Logger {
	if logger, err := slog.NewLoggerFromCfgFile(`log_scheduler`); err == nil {
		return logger.With(slog.NewTagField(`scheduler`))
	}

	return slog.NewLogger(`scheduler`)
}

type Logger struct {
	logger *zap.Logger
}

func newSchedulerLogger() cron.Logger {
	return &Logger{logger: newSchedulerZapLogger()}
}

func (l Logger) Info(msg string, keysAndValues ...interface{}) {
	l.logger.Debug(formatLog(msg, keysAndValues...)) //降级，主要显示冗余日志
}

func (l Logger) Error(err error, msg string, keysAndValues ...interface{}) {
	l.logger.Error(formatLog(msg, keysAndValues...), zap.Error(err))
}

func formatLog(msg string, keysAndValues ...interface{}) string {
	if len(keysAndValues) == 0 {
		return msg
	}

	var sb strings.Builder
	sb.WriteString(msg)
	sb.WriteString("[")

	for i, val := range keysAndValues {
		if i%2 == 0 {
			if i > 0 {
				sb.WriteString(`,`)
			}

			sb.WriteString(fmt.Sprintf("%v", val))
			continue
		}

		if t, ok := val.(time.Time); ok {
			val = t.Format(time.RFC3339)
		}

		sb.WriteString(fmt.Sprintf("=%v", val))
	}

	sb.WriteString("]")
	return sb.String()
}
