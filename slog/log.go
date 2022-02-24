package slog

import (
	"fmt"
	"github.com/bingooh/b-go-util/conf"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"log"
	"os"
	"strings"
)

var rootLogger = NewDebugLogger(``, zapcore.DebugLevel)

//过滤日志标签，返回true则不显示日志。仅用于调试！
//如果需要过滤日志消息，建议做法应该是定制zapcore.Encoder,过滤日志field
var logTagFilter = func(tag string) bool {
	return false
}

func SetLogTagFilter(filter func(tag string) bool) {
	if filter != nil {
		logTagFilter = filter
		log.Println(`SetLogTagFilter()仅用于调试时过滤日志，请勿用于正式环境`)
	}
}

func NewTagField(tags ...string) zap.Field {
	return zap.String(LogTagFieldName, strings.Join(tags, `.`))
}

func NewConfFromFile(cfgFilePath string) (cfg *Conf, err error) {
	cfg = &Conf{}
	err = conf.ScanConfFile(cfg, cfgFilePath)

	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func NewLoggerFromCfg(cfg Conf) *zap.Logger {
	cfg = cfg.Normalize()

	level := zap.NewAtomicLevelAt(cfg.Level)

	//encoder -> writer -> core
	//zap是对zapcore的封装，如果要输出到多个core，可使用zapcore.NewTee()
	encoderCfg := zap.NewDevelopmentEncoderConfig()
	if !cfg.DisableColor {
		encoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	if !cfg.EnableShortCaller {
		encoderCfg.EncodeCaller = zapcore.FullCallerEncoder
	}

	var encoder zapcore.Encoder
	if cfg.Encoding == `json` {
		encoder = zapcore.NewJSONEncoder(encoderCfg)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	}

	//如果writer不支持并发访问，必须使用锁独占writer
	//文件是不支持并发写入的。见zapcore.Lock()，zap.CombineWriteSyncers()
	var writers []zapcore.WriteSyncer

	if cfg.WriteToConsole {
		writers = append(writers, zapcore.AddSync(os.Stdout))
	}

	if cfg.WriteToLogFile {
		w := zapcore.AddSync(&lumberjack.Logger{
			Filename:   cfg.LogFilePath,
			MaxSize:    cfg.LogFileMaxSize,
			MaxBackups: cfg.LogFileMaxBackups,
			MaxAge:     cfg.LogFileMaxAge,
			Compress:   cfg.CompressLogFile,
		})

		writers = append(writers, w)
	}

	var options []zap.Option
	if cfg.Debug {
		options = append(options, zap.Development())
	}

	if !cfg.DisableCaller {
		options = append(options, zap.AddCaller(), zap.AddCallerSkip(cfg.CallerSkip))
	}

	if cfg.EnableStackTrace {
		options = append(options, zap.AddStacktrace(level))
	}

	core := zapcore.NewCore(encoder, zapcore.NewMultiWriteSyncer(writers...), level)
	return zap.New(core, options...)
}

func NewLoggerFromCfgFile(cfgFilePath string) (*zap.Logger, error) {
	cfg, err := NewConfFromFile(cfgFilePath)
	if err != nil {
		return nil, err
	}

	return NewLoggerFromCfg(*cfg), nil
}

func RootLogger() *zap.Logger {
	return rootLogger
}

//使用自定义的rootLogger
func InitRootLogger(logger *zap.Logger) {
	if logger == nil {
		panic(`logger is nil`)
	}

	rootLogger = logger
}

//使用自定义的rootLogger
func InitRootLoggerFromCfg(cfg Conf) {
	InitRootLogger(NewLoggerFromCfg(cfg))
}

//初始化默认日志组件，建议在主程序启动前调用此方法
func MustInitDefaultRootLogger() {
	cfg, err := NewConfFromFile(LogConfFileName)
	if err != nil {
		cfg = &Conf{WriteToConsole: true}
		log.Printf("读取默认日志配置文件出错,将使用默认日志配置[%v]\n", err)
	}

	rootLogger = NewLoggerFromCfg(*cfg)
}

//清空缓存的日志，此方法应在主程序退出前调用
func Flush() error {
	if rootLogger != nil {
		return rootLogger.Sync()
	}

	return nil
}

//必须先初始化rootLogger，否则调用此方法将抛出空指针错误
func NewLogger(tags ...string) *zap.Logger {
	tag := strings.Join(tags, `.`)

	if logTagFilter(tag) {
		return zap.NewNop()
	}

	return rootLogger.With(NewTagField(tags...))
}

//用于调试的日志器
func NewDebugLogger(tag string, level zapcore.Level) *zap.Logger {
	c := zap.NewDevelopmentConfig()

	c.Level = zap.NewAtomicLevelAt(level)
	c.DisableStacktrace = true
	c.EncoderConfig.EncodeCaller = zapcore.FullCallerEncoder
	c.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	logger, err := c.Build()
	if err != nil {
		panic(fmt.Errorf(`create debug logger err[%v]`, err))
	}

	if tag == "" {
		return logger
	}

	return logger.With(NewTagField(tag))
}
