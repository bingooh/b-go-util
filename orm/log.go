package orm

import (
	"b-go-util/slog"
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	glog "gorm.io/gorm/logger"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

//创建用于输出db日志的日志器，默认读取conf/log_db配置文件，如无则调用slog.NewLogger()
func newDBZapLogger() *zap.Logger {
	if logger, err := slog.NewLoggerFromCfgFile(`log_db`); err == nil {
		return logger.With(slog.NewTagField(`db`))
	}

	return slog.NewLogger(`db`)
}

//输出日志到std，仅用于调试
func newDBStdLogger(option LoggerOption) glog.Interface {
	return glog.New(
		log.New(os.Stdout, "", log.LstdFlags),
		glog.Config{
			Colorful:                  true,
			LogLevel:                  option.LogLevel,
			SlowThreshold:             option.SlowThreshold,
			IgnoreRecordNotFoundError: option.IgnoreRecordNotFoundError,
		})
}

//日志选项，参考glog.Config
type LoggerOption struct {
	//Debug                     bool        //是否启用调试，如果为true，则将使用std日志器
	LogLevel                  glog.LogLevel //日志级别
	SlowThreshold             time.Duration //慢SQL耗时临界值
	IgnoreRecordNotFoundError bool          //是否不输出查询结果为空错误日志
}

//实现gorm的日志接口，底层使用slog输出日志
type Logger struct {
	option LoggerOption
	logger *zap.Logger
	level  glog.LogLevel
}

//输出SQL日志
func newDBLogger(option LoggerOption) glog.Interface {
	/*if option.Debug{
		return newDBStdLogger(option)
	}*/

	return &Logger{
		option: option,
		logger: newDBZapLogger(),
		level:  option.LogLevel,
	}
}

func (l Logger) LogMode(level glog.LogLevel) glog.Interface {
	return Logger{
		option: l.option,
		logger: l.logger,
		level:  level,
	}
}

func (l Logger) Info(ctx context.Context, s string, i ...interface{}) {
	if l.level >= glog.Info {
		l.loggerWithSkippedCaller().Sugar().Infof(s, i...)
	}
}

func (l Logger) Warn(ctx context.Context, s string, i ...interface{}) {
	if l.level >= glog.Warn {
		l.loggerWithSkippedCaller().Sugar().Warnf(s, i...)
	}
}

func (l Logger) Error(ctx context.Context, s string, i ...interface{}) {
	if l.level >= glog.Error {
		l.loggerWithSkippedCaller().Sugar().Errorf(s, i...)
	}
}

func (l Logger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.level <= glog.Silent {
		return
	}

	elapsed := time.Since(begin)
	switch {
	case err != nil && l.level >= glog.Error && (!errors.Is(err, glog.ErrRecordNotFound) || !l.option.IgnoreRecordNotFoundError):
		sql, rows := fc()
		l.loggerWithSkippedCaller().Error(`trace`, zap.Error(err), zap.Duration("elapsed", elapsed), zap.Int64("rows", rows), zap.String("sql", sql))
	case elapsed > l.option.SlowThreshold && l.option.SlowThreshold != 0 && l.level >= glog.Warn:
		sql, rows := fc()
		msg := fmt.Sprintf(`trace slow log(>= %v)`, l.option.SlowThreshold)
		l.loggerWithSkippedCaller().Warn(msg, zap.Error(err), zap.Duration("elapsed", elapsed), zap.Int64("rows", rows), zap.String("sql", sql))
	case l.level == glog.Info:
		sql, rows := fc()
		l.loggerWithSkippedCaller().Info(`trace`, zap.Error(err), zap.Duration("elapsed", elapsed), zap.Int64("rows", rows), zap.String("sql", sql))
	}
}

//需要跳过的日志调用者路径
const (
	gormPkg = `gorm.io/gorm`
	bormPkg = `b-go-util/orm`
)

func (l Logger) loggerWithSkippedCaller() *zap.Logger {
	for i := 2; i < 15; i++ {
		_, file, _, ok := runtime.Caller(i)
		switch {
		case !ok:
		case strings.Contains(file, gormPkg):
		case strings.Contains(file, bormPkg):
		default:
			return l.logger.WithOptions(zap.AddCallerSkip(i - 1))
		}
	}
	return l.logger
}
